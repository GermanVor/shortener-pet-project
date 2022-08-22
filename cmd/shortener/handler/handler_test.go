package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GermanVor/shortener-pet-project/cmd/shortener/handler"
	"github.com/GermanVor/shortener-pet-project/storage"
	"github.com/bmizerany/assert"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

func createTestEnvironment() (*storage.Storage, string, func()) {
	router := gin.Default()

	storage := storage.InitStorage()

	handler.InitShortenerHandlers(router, storage)
	ts := httptest.NewServer(router)

	endpointURL := ts.URL

	cleanupFunc := func() {
		ts.Close()
	}

	return storage, endpointURL, cleanupFunc
}

var errRedirectBlocked = errors.New("HTTP redirect blocked")
var redirPolicy = resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
	return errRedirectBlocked
})

func TestServerOperations(t *testing.T) {
	storage, endpointURL, cleanupFunc := createTestEnvironment()
	defer cleanupFunc()

	t.Run("V1", func(t *testing.T) {
		originalURL := "http://oknetcumk.biz/" + t.Name()
		shortURL := ""

		{
			bodyReader := bytes.NewReader([]byte(originalURL))
			req, err := http.NewRequest(http.MethodPost, endpointURL, bodyReader)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			shortURL = string(bodyBytes)

			t.Log("shortURL from Server: ", shortURL)

			resp.Body.Close()
		}

		{
			req := resty.New().
				SetRedirectPolicy(redirPolicy).
				R()

			resp, _ := req.Get(shortURL)

			assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode())
			assert.Equal(t, originalURL, resp.Header().Get("Location"))
		}

		t.Log(storage)
	})

	t.Run("V2", func(t *testing.T) {
		originalURL := "http://oknetcumk.biz/" + t.Name()

		requestBody := &handler.MakeShortPostEndpointRequest{
			URL: originalURL,
		}
		requestBytes, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, endpointURL+"/api/shorten", bytes.NewReader(requestBytes))
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		responseBody := &handler.MakeShortPostEndpointResponse{}
		err = json.NewDecoder(resp.Body).Decode(responseBody)
		require.NoError(t, err)

		t.Log(responseBody)

		{
			req := resty.New().
				SetRedirectPolicy(redirPolicy).
				R()

			resp, _ := req.Get(responseBody.Result)

			assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode())
			assert.Equal(t, originalURL, resp.Header().Get("Location"))
		}
	})
}

package handler_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/GermanVor/shortener-pet-project/cmd/shortener/handler"
	"github.com/GermanVor/shortener-pet-project/internal/storage"
	"github.com/bmizerany/assert"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var (
	endpointURL = "http://127.0.0.1:8080"
)

func TestShortenURLV1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	storage := storage.Init(endpointURL, "")
	handler.InitShortenerHandlers(router, storage)

	originalURL := "http://oknetcumk.biz/" + t.Name()
	shortURL := ""

	{
		bodyReader := bytes.NewReader([]byte(originalURL))
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/", bodyReader)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		shortURL = string(bodyBytes)
	}

	{
		req, err := http.NewRequest(http.MethodGet, shortURL, nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Equal(t, originalURL, resp.Header.Get("Location"))
	}

	{
		request := handler.MakeShortPostEndpointRequest{
			URL: originalURL,
		}
		bytesRequest, err := json.Marshal(request)
		require.NoError(t, err)

		bodyReader := bytes.NewReader(bytesRequest)
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/api/shorten", bodyReader)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		respObj := handler.MakeShortPostEndpointResponse{}
		json.Unmarshal(bodyBytes, &respObj)

		assert.Equal(t, shortURL, respObj.Result)
	}
}

func TestShortenURLV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	storage := storage.Init(endpointURL, "")
	handler.InitShortenerHandlers(router, storage)

	originalURL := "http://oknetcumk.biz/" + t.Name()
	shortURL := ""

	{
		request := handler.MakeShortPostEndpointRequest{
			URL: originalURL,
		}
		bytesRequest, err := json.Marshal(request)
		require.NoError(t, err)

		bodyReader := bytes.NewReader(bytesRequest)
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/api/shorten", bodyReader)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		respObj := handler.MakeShortPostEndpointResponse{}
		json.Unmarshal(bodyBytes, &respObj)

		shortURL = respObj.Result
	}

	{
		req, err := http.NewRequest(http.MethodGet, shortURL, nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Equal(t, originalURL, resp.Header.Get("Location"))
	}
}

func TestMiddlwareV1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	storage := storage.Init(endpointURL, "")
	handler.InitShortenerHandlers(router, storage)

	originalURL := "http://oknetcumk.biz/" + t.Name()
	shortURL := ""

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(originalURL))
	_ = zw.Close()

	{
		bodyReader := bytes.NewReader(buf.Bytes())
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/", bodyReader)
		require.NoError(t, err)

		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Encoding", "gzip")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		shortURL = string(bodyBytes)
	}

	{
		_, urlParseErr := url.Parse(shortURL)

		t.Log(urlParseErr)

		req, err := http.NewRequest(http.MethodGet, shortURL, nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Equal(t, originalURL, resp.Header.Get("Location"))
	}
}

func TestMiddlwareV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	storage := storage.Init(endpointURL, "")
	handler.InitShortenerHandlers(router, storage)

	originalURL := "http://oknetcumk.biz/" + t.Name()
	shortURL := ""

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(originalURL))
	_ = zw.Close()

	{
		bodyReader := bytes.NewReader(buf.Bytes())
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/", bodyReader)
		require.NoError(t, err)

		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Encoding", "gzip")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		shortURL = string(bodyBytes)
	}

	{
		_, urlParseErr := url.Parse(shortURL)

		t.Log(urlParseErr)

		req, err := http.NewRequest(http.MethodGet, shortURL, nil)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Equal(t, originalURL, resp.Header.Get("Location"))
	}
}

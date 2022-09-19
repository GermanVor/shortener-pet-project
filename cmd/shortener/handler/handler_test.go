package handler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/GermanVor/shortener-pet-project/cmd/shortener/handler"
	"github.com/GermanVor/shortener-pet-project/internal/storage"
	"github.com/bmizerany/assert"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
)

var (
	endpointURL = "http://127.0.0.1:8080"
	connString  = "postgres://zzman:@localhost:5432/test"
)

func CleanDB() {
	conn, err := pgxpool.Connect(context.TODO(), connString)
	if err != nil {
		log.Fatalln(err)
	}

	sql := "DROP TABLE IF EXISTS shortensArchive;"
	_, err = conn.Exec(context.TODO(), sql)
	if err != nil {
		log.Fatalln(err)
	}

	sql = "DROP TABLE IF EXISTS usersArchive;"
	_, err = conn.Exec(context.TODO(), sql)
	if err != nil {
		log.Fatalln(err)
	}

	conn.Close()
}

func TestShortenURLV1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := func(tt *testing.T, stor storage.Interface) {
		router := gin.Default()
		handler.InitShortenerHandlers(router, stor)

		originalURL := "http://oknetcumk.biz/" + tt.Name()
		shortURL := ""

		{
			bodyReader := bytes.NewReader([]byte(originalURL))
			req, err := http.NewRequest(http.MethodPost, endpointURL+"/", bodyReader)
			require.NoError(tt, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			resp := recorder.Result()
			defer resp.Body.Close()

			require.Equal(tt, http.StatusCreated, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(tt, err)
			shortURL = string(bodyBytes)
		}

		{
			req, err := http.NewRequest(http.MethodGet, shortURL, nil)
			require.NoError(tt, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			resp := recorder.Result()
			defer resp.Body.Close()

			assert.Equal(tt, http.StatusTemporaryRedirect, resp.StatusCode)
			assert.Equal(tt, originalURL, resp.Header.Get("Location"))
		}

		{
			request := handler.MakeShortPostEndpointRequest{
				URL: originalURL,
			}
			bytesRequest, err := json.Marshal(request)
			require.NoError(tt, err)

			bodyReader := bytes.NewReader(bytesRequest)
			req, err := http.NewRequest(http.MethodPost, endpointURL+"/api/shorten", bodyReader)
			require.NoError(tt, err)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			resp := recorder.Result()
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(tt, err)

			respObj := handler.MakeShortPostEndpointResponse{}
			json.Unmarshal(bodyBytes, &respObj)

			assert.Equal(tt, shortURL, respObj.Result)
		}
	}

	t.Run("Storage mock", func(tt *testing.T) {
		stor := storage.InitV1(endpointURL, "")

		testBody(tt, stor)
	})

	// t.Run("Storage DB", func(tt *testing.T) {
	// 	CleanDB()

	// 	dbContext := context.Background()
	// 	stor, err := storage.InitV2(endpointURL, dbContext, connString)
	// 	require.NoError(tt, err)

	// 	testBody(tt, stor)
	// })
}

func TestShortenURLV2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	storage := storage.InitV1(endpointURL, "")
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
	storage := storage.InitV1(endpointURL, "")
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
	storage := storage.InitV1(endpointURL, "")
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

func TestMiddlware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	storage := storage.InitV1(endpointURL, "")

	router := gin.Default()
	router.Use(handler.UseCookieMiddlware)

	handler.InitShortenerHandlers(router, storage)

	originalURL := "http://oknetcumk.biz/1"
	shortURL := ""

	cookie := &http.Cookie{
		Name:  handler.SessionTokenName,
		Value: "some_token",
	}

	{
		bodyReader := bytes.NewReader([]byte(originalURL))
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/", bodyReader)
		require.NoError(t, err)

		req.AddCookie(cookie)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		shortURL = string(bodyBytes)
	}

	{
		req, err := http.NewRequest(http.MethodGet, endpointURL+"/api/user/urls", nil)
		require.NoError(t, err)
		req.AddCookie(cookie)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		resp := recorder.Result()
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var urls []handler.UserUrls

		err = json.Unmarshal(bodyBytes, &urls)
		require.NoError(t, err)

		assert.Equal(t, 1, len(urls))
		assert.Equal(t, originalURL, urls[0].OriginalURL)
		assert.Equal(t, shortURL, urls[0].ShortURL)
	}
}

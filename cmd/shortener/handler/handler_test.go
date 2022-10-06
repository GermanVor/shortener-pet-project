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

func CheckRedirect(t *testing.T, shortURL, originalURL string, serveHTTP func(w http.ResponseWriter, req *http.Request)) {
	req, err := http.NewRequest(http.MethodGet, shortURL, nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	serveHTTP(recorder, req)
	resp := recorder.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, originalURL, resp.Header.Get("Location"))
}

func TestShortenURLV1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := func(tt *testing.T, stor storage.Interface) {
		router := gin.Default()
		handler.InitShortenerHandlers(router, stor)

		originalURL := "qwe"
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

		CheckRedirect(tt, shortURL, originalURL, router.ServeHTTP)

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

			require.Equal(tt, http.StatusConflict, resp.StatusCode)

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
	// 	defer CleanDB()

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

	CheckRedirect(t, shortURL, originalURL, router.ServeHTTP)
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

	CheckRedirect(t, shortURL, originalURL, router.ServeHTTP)
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

	CheckRedirect(t, shortURL, originalURL, router.ServeHTTP)
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

func TestMakeShortsPostEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	storage := storage.InitV1(endpointURL, "")

	router := gin.Default()
	router.Use(handler.UseCookieMiddlware)

	handler.InitShortenerHandlers(router, storage)

	requestBody := []handler.MakeShortsPostEndpointRequest{
		{CorrelationID: "qwe", OriginalURL: "http://oknetcumk.biz/1"},
	}

	{
		requestBytes, _ := json.Marshal(requestBody)
		bodyReader := bytes.NewReader(requestBytes)
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/api/shorten/batch", bodyReader)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		resp := recorder.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		responseBody := []handler.MakeShortsPostEndpointResponse{}
		require.NoError(t, json.Unmarshal(bodyBytes, &responseBody))

		require.Equal(t, 1, len(responseBody))

		assert.Equal(t, requestBody[0].CorrelationID, responseBody[0].CorrelationID)

		CheckRedirect(t, responseBody[0].ShortURL, requestBody[0].OriginalURL, router.ServeHTTP)
	}
}

func TestDeleteEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := func(tt *testing.T, stor storage.Interface) {
		router := gin.Default()
		router.Use(handler.UseCookieMiddlware)

		handler.InitShortenerHandlers(router, stor)

		originalURLs := []string{"qwe", "rty"}
		shortURLsID := make([]string, len(originalURLs))

		cookie := &http.Cookie{
			Name:  handler.SessionTokenName,
			Value: "some_token",
		}

		for i, originalURL := range originalURLs {
			bodyReader := bytes.NewReader([]byte(originalURL))
			req, err := http.NewRequest(http.MethodPost, endpointURL+"/", bodyReader)
			require.NoError(tt, err)

			req.AddCookie(cookie)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			resp := recorder.Result()

			require.Equal(tt, http.StatusCreated, resp.StatusCode)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(tt, err)

			shortURLsID[i] = string(bodyBytes)[len(endpointURL)+1:]

			resp.Body.Close()
		}

		{
			requestBytes, _ := json.Marshal(shortURLsID)
			bodyReader := bytes.NewReader(requestBytes)
			req, err := http.NewRequest(http.MethodDelete, endpointURL+"/api/user/urls", bodyReader)
			require.NoError(t, err)

			req.AddCookie(cookie)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			resp := recorder.Result()
			defer resp.Body.Close()

			assert.Equal(t, http.StatusAccepted, resp.StatusCode)
		}

		for _, shortURLId := range shortURLsID {
			req, err := http.NewRequest(http.MethodGet, endpointURL+"/"+shortURLId, nil)
			require.NoError(t, err)

			req.AddCookie(cookie)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			resp := recorder.Result()
			resp.Body.Close()

			assert.Equal(t, http.StatusGone, resp.StatusCode)
		}
	}

	t.Run("Storage mock", func(tt *testing.T) {
		stor := storage.InitV1(endpointURL, "")

		testBody(tt, stor)
	})

	// t.Run("Storage DB", func(tt *testing.T) {
	// 	CleanDB()
	// 	// defer CleanDB()

	// 	dbContext := context.Background()
	// 	stor, err := storage.InitV2(endpointURL, dbContext, connString)
	// 	require.NoError(tt, err)

	// 	testBody(tt, stor)
	// })
}

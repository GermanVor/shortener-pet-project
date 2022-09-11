package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/GermanVor/shortener-pet-project/internal/storage"
	"github.com/gin-gonic/gin"
)

func MakeShortEndpoint(ctx *gin.Context, shortenURL func(string) string) {
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.Fatal(err)
	}

	originalURL := string(bodyBytes)
	shortURL := shortenURL(originalURL)

	ctx.Writer.Header().Set("Content-Type", "text/plain")
	ctx.Writer.WriteHeader(http.StatusCreated)
	ctx.Writer.Write([]byte(shortURL))
}

func GetFullStrEndpoint(ctx *gin.Context, getOriginalURL func(shortURL string) (string, bool)) {
	shortURL := ctx.Param("id")

	if shortURL == "" {
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if originalURL, ok := getOriginalURL(shortURL); ok {
		ctx.Writer.Header().Set("Location", originalURL)
		ctx.Writer.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		ctx.Writer.WriteHeader(http.StatusBadRequest)
	}
}

type MakeShortPostEndpointRequest struct {
	URL string `json:"url"`
}
type MakeShortPostEndpointResponse struct {
	Result string `json:"result"`
}

func MakeShortPostEndpoint(ctx *gin.Context, shortenURL func(string) string) {
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.Fatal(err)
	}

	request := &MakeShortPostEndpointRequest{}
	err = json.Unmarshal(bodyBytes, request)
	if err != nil {
		log.Fatal(err)
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	respose := &MakeShortPostEndpointResponse{
		Result: shortenURL(request.URL),
	}
	responseBytes, _ := json.Marshal(respose)

	ctx.Writer.Header().Set("Content-Type", "application/json")
	ctx.Writer.WriteHeader(http.StatusCreated)
	ctx.Writer.Write(responseBytes)
}

func InitShortenerHandlers(router *gin.Engine, storage *storage.Storage) *gin.Engine {
	router.POST("/", func(ctx *gin.Context) {
		MakeShortEndpoint(ctx, storage.ShortenURL)
	})

	router.GET("/:id", func(ctx *gin.Context) {
		GetFullStrEndpoint(ctx, storage.GetOriginalURL)
	})

	router.POST("/api/shorten", func(ctx *gin.Context) {
		MakeShortPostEndpoint(ctx, storage.ShortenURL)
	})

	return router
}

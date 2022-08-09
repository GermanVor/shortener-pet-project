package handler

import (
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MakeShortEndpoint(ctx *gin.Context, shortenUrl func(string) string) {
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.Fatal(err)
	}

	originalUrl := string(bodyBytes)
	shortUrl := shortenUrl(originalUrl)

	ctx.Writer.Header().Set("Content-Type", "text/plain")
	ctx.Writer.WriteHeader(http.StatusCreated)
	ctx.Writer.Write([]byte("http://" + ctx.Request.Host + "/" + shortUrl))
}

func GetFullStrEndpoint(ctx *gin.Context, getOriginalUrl func(shortUrl string) string) {
	shortUrl := ctx.Param("id")

	if shortUrl == "" {
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	originalUrl := getOriginalUrl(shortUrl)

	if originalUrl == "" {
		ctx.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx.Writer.Header().Set("Location", originalUrl)
	ctx.Writer.WriteHeader(http.StatusTemporaryRedirect)
}

type Storage interface {
	GetOriginalUrl(a string) string
	ShortenUrling(b string) string
}

func InitShortenerHandlers(router *gin.Engine, storage Storage) *gin.Engine {
	router.POST("/", func(ctx *gin.Context) {
		MakeShortEndpoint(ctx, storage.ShortenUrling)
	})

	router.GET("/:id", func(ctx *gin.Context) {
		GetFullStrEndpoint(ctx, storage.GetOriginalUrl)
	})

	return router
}

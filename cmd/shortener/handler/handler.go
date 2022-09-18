package handler

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/GermanVor/shortener-pet-project/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var SessionTokenName = "session_token"

func MakeShortEndpoint(ctx *gin.Context, stor storage.Interface) {
	r := ctx.Request
	w := ctx.Writer

	originalURL := ""
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		gReader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
			return
		}

		bytes, _ := io.ReadAll(gReader)
		originalURL = string(bytes)
	} else {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(ctx.Writer, err.Error(), http.StatusInternalServerError)
			return
		}

		originalURL = string(bodyBytes)
	}

	shortURL := stor.ShortenURL(originalURL, ctx.GetString(SessionTokenName))

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func GetFullStrEndpoint(ctx *gin.Context, stor storage.Interface) {
	w := ctx.Writer

	shortURL := ctx.Param("id")

	if shortURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if originalURL, ok := stor.GetOriginalURL(shortURL); ok {
		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

type MakeShortPostEndpointRequest struct {
	URL string `json:"url"`
}
type MakeShortPostEndpointResponse struct {
	Result string `json:"result"`
}

func MakeShortPostEndpoint(ctx *gin.Context, stor storage.Interface) {
	w := ctx.Writer
	r := ctx.Request

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	request := &MakeShortPostEndpointRequest{}
	err = json.Unmarshal(bodyBytes, request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respose := &MakeShortPostEndpointResponse{
		Result: stor.ShortenURL(request.URL, ctx.GetString(SessionTokenName)),
	}
	responseBytes, _ := json.Marshal(respose)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(responseBytes)
}

type UserUrls = storage.UserUrls

func GetUsersArchiveEndpoint(ctx *gin.Context, stor storage.Interface) {
	w := ctx.Writer

	userToken := ctx.GetString(SessionTokenName)
	archive, ok := stor.GetUserArchive(userToken)

	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	responseBytes, _ := json.Marshal(archive)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func InitShortenerHandlers(router *gin.Engine, stor storage.Interface) *gin.Engine {
	router.Use(func(ctx *gin.Context) {
		cookie, err := ctx.Request.Cookie(SessionTokenName)
		if err != nil {
			log.Println(err)
		}

		if cookie == nil {
			cookie = &http.Cookie{
				Name:  SessionTokenName,
				Value: uuid.NewString(),
			}

			http.SetCookie(ctx.Writer, cookie)
		}

		ctx.Set(SessionTokenName, cookie.Value)
		ctx.Next()
	})

	router.POST("/", func(ctx *gin.Context) {
		MakeShortEndpoint(ctx, stor)
	})

	router.GET("/:id", func(ctx *gin.Context) {
		GetFullStrEndpoint(ctx, stor)
	})

	router.POST("/api/shorten", func(ctx *gin.Context) {
		MakeShortPostEndpoint(ctx, stor)
	})

	router.GET("/api/user/urls", func(ctx *gin.Context) {
		GetUsersArchiveEndpoint(ctx, stor)
	})

	return router
}

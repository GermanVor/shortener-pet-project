package handler

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/GermanVor/shortener-pet-project/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserUrls = storage.UserUrls

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

	shortURL, err := stor.ShortenURL(originalURL, ctx.GetString(SessionTokenName))

	if err == storage.ErrValueAlreadyShorted {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(shortURL))
}

func GetFullStrEndpoint(ctx *gin.Context, stor storage.Interface) {
	w := ctx.Writer

	shortURL := ctx.Param("id")

	if shortURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	originalURL, err := stor.GetOriginalURL(shortURL, ctx.GetString(SessionTokenName))

	if err != nil {
		if errors.Is(err, storage.ErrValueNotFound) {
			w.WriteHeader(http.StatusBadRequest)
		} else if errors.Is(err, storage.ErrValueGone) {
			w.WriteHeader(http.StatusGone)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
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

	shortURL, err := stor.ShortenURL(request.URL, ctx.GetString(SessionTokenName))

	respose := &MakeShortPostEndpointResponse{
		Result: shortURL,
	}
	responseBytes, _ := json.Marshal(respose)

	if err == storage.ErrValueAlreadyShorted {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

type MakeShortsPostEndpointRequest = storage.MappingItem
type MakeShortsPostEndpointResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func MakeShortsPostEndpoint(ctx *gin.Context, stor storage.Interface) {
	w := ctx.Writer
	r := ctx.Request

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req := []MakeShortsPostEndpointRequest{}
	err = json.Unmarshal(bodyBytes, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]MakeShortsPostEndpointResponse, 0)

	stor.ForEach(req, ctx.GetString(SessionTokenName), func(correlationID, shortURL string) error {
		resp = append(resp, MakeShortsPostEndpointResponse{
			CorrelationID: correlationID,
			ShortURL:      shortURL,
		})

		return nil
	})

	responseBytes, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(responseBytes)
}

func DeleteUrls(ctx *gin.Context, stor storage.Interface) {
	w := ctx.Writer
	r := ctx.Request

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	keys := []string{}
	err = json.Unmarshal(bodyBytes, &keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = stor.DeleteKeys(keys, ctx.GetString(SessionTokenName))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func GetUsersArchiveEndpoint(ctx *gin.Context, stor storage.Interface) {
	w := ctx.Writer

	userToken := ctx.GetString(SessionTokenName)
	archive, err := stor.GetUserArchive(userToken)

	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	responseBytes, _ := json.Marshal(archive)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func UseCookieMiddlware(ctx *gin.Context) {
	cookie, err := ctx.Request.Cookie(SessionTokenName)
	if err != nil {
		log.Println(err)
	}

	if cookie == nil && ctx.Request.Method == http.MethodPost {
		cookie = &http.Cookie{
			Name:  SessionTokenName,
			Value: uuid.NewString(),
		}

		http.SetCookie(ctx.Writer, cookie)
	}

	if cookie != nil {
		ctx.Set(SessionTokenName, cookie.Value)
	}

	ctx.Next()
}

func InitShortenerHandlers(router *gin.Engine, stor storage.Interface) *gin.Engine {
	router.POST("/", func(ctx *gin.Context) {
		MakeShortEndpoint(ctx, stor)
	})

	router.POST("/api/shorten", func(ctx *gin.Context) {
		MakeShortPostEndpoint(ctx, stor)
	})

	router.POST("/api/shorten/batch", func(ctx *gin.Context) {
		MakeShortsPostEndpoint(ctx, stor)
	})

	router.GET("/:id", func(ctx *gin.Context) {
		GetFullStrEndpoint(ctx, stor)
	})

	router.GET("/api/user/urls", func(ctx *gin.Context) {
		GetUsersArchiveEndpoint(ctx, stor)
	})

	router.DELETE("/api/user/urls", func(ctx *gin.Context) {
		DeleteUrls(ctx, stor)
	})

	return router
}

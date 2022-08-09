package main

import (
	handler "github.com/GermanVor/shortener-pet-project/cmd/shortener/handler"
	"github.com/GermanVor/shortener-pet-project/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	storage := storage.InitStorage()
	router := gin.Default()

	handler.InitShortenerHandlers(router, storage)

	router.Run(":8080")
}

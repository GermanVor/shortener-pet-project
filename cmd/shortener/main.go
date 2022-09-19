package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	handler "github.com/GermanVor/shortener-pet-project/cmd/shortener/handler"
	common "github.com/GermanVor/shortener-pet-project/internal/common"
	"github.com/GermanVor/shortener-pet-project/internal/storage"
	"github.com/gin-gonic/gin"
)

var Config = &common.Config{
	ServerAddress:   "localhost:8080",
	BaseURL:         "http://localhost:8080",
	FileStoragePath: "",
}

func initConfig() {
	common.InitFlagsConfig(Config)
	flag.Parse()
	common.InitEnvConfig(Config)

	log.Println("Config", Config)
}

func main() {
	initConfig()

	router := gin.Default()
	router.Use(handler.UseCookieMiddlware)

	var stor storage.Interface

	if Config.DatabaseDSN != "" {
		dbContext := context.Background()
		storV2, err := storage.InitV2(Config.BaseURL, dbContext, Config.DatabaseDSN)
		if err != nil {
			log.Fatalln(err)
		}

		router.GET("ping", func(ctx *gin.Context) {
			if storV2.Ping() == nil {
				ctx.Writer.WriteHeader(http.StatusOK)
			}
		})

		stor = storV2
	} else {
		stor = storage.InitV1(Config.BaseURL, Config.FileStoragePath)
	}

	handler.InitShortenerHandlers(router, stor)

	log.Println("Server started at", Config.ServerAddress)

	err := router.Run(Config.ServerAddress)
	if err != nil {
		log.Println("Server could not start", err)
	}
}

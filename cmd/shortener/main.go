package main

import (
	"flag"
	"log"

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

	stor := storage.Init(Config.BaseURL, Config.FileStoragePath)
	router := gin.Default()

	handler.InitShortenerHandlers(router, stor)

	log.Println("Server started at", Config.ServerAddress)

	err := router.Run(Config.ServerAddress)
	if err != nil {
		log.Println("Server could not start", err)
	}
}

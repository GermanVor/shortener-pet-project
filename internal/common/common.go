package common

import (
	"flag"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
}

func InitEnvConfig(config *Config) *Config {
	godotenv.Load(".env")

	if serverAddress, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		config.ServerAddress = serverAddress
	}

	if baseURL, ok := os.LookupEnv("BASE_URL"); ok {
		config.BaseURL = baseURL
	}

	if fileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		config.FileStoragePath = fileStoragePath
	}

	return config
}

const (
	aUsage = "Address"
	bUsage = "Base URL"
	fUsage = "Storage file path"
)

func InitFlagsConfig(config *Config) *Config {
	flag.StringVar(&config.ServerAddress, "a", config.ServerAddress, aUsage)
	flag.StringVar(&config.BaseURL, "b", config.BaseURL, bUsage)
	flag.StringVar(&config.FileStoragePath, "f", config.FileStoragePath, fUsage)

	return config
}

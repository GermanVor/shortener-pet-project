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

	DatabaseDSN string
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

	if datavaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		config.DatabaseDSN = datavaseDSN
	}

	return config
}

const (
	aUsage = "Address"
	bUsage = "Base URL"
	fUsage = "Storage file path"
	dUsage = "Database address to connect"
)

func InitFlagsConfig(config *Config) *Config {
	flag.StringVar(&config.ServerAddress, "a", config.ServerAddress, aUsage)
	flag.StringVar(&config.BaseURL, "b", config.BaseURL, bUsage)
	flag.StringVar(&config.FileStoragePath, "f", config.FileStoragePath, fUsage)
	flag.StringVar(&config.DatabaseDSN, "d", config.DatabaseDSN, dUsage)

	return config
}

func min[T ~int](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Chunks[T any](arr []T, chunkSize int) [][]T {
	if len(arr) == 0 || chunkSize <= 0 {
		return nil
	}

	chunks := make([][]T, (len(arr)-1)/chunkSize+1)

	for i := range chunks {
		leftIdx := i * chunkSize
		rightIdx := leftIdx + min(chunkSize, len(arr)-leftIdx)

		chunks[i] = arr[leftIdx:rightIdx:rightIdx]
	}

	return chunks
}

package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type Storage struct {
	db     map[string]string
	keysDB map[string]string
	dbMux  sync.RWMutex

	baseURL         string
	fileStoragePath string
}

func (s *Storage) ShortenURL(originalURL string) string {
	s.dbMux.Lock()
	defer s.dbMux.Unlock()

	shortURL, isAlreadySaved := s.keysDB[originalURL]

	if !isAlreadySaved {
		shortURL = strconv.Itoa(len(s.db) + 1)

		s.keysDB[originalURL] = shortURL
		s.db[shortURL] = originalURL

		if s.fileStoragePath != "" {
			backupBytes, _ := json.Marshal(&s.db)
			os.WriteFile(s.fileStoragePath, backupBytes, 0644)
		}
	}

	return s.baseURL + "/" + shortURL
}

func (s *Storage) GetOriginalURL(shortURL string) (string, bool) {
	s.dbMux.RLock()
	originalURL, ok := s.db[shortURL]
	s.dbMux.RUnlock()

	return originalURL, ok
}

func Init(baseURL, fileStoragePath string) *Storage {
	s := &Storage{
		db:              make(map[string]string),
		keysDB:          make(map[string]string),
		baseURL:         baseURL,
		fileStoragePath: fileStoragePath,
	}

	if fileStoragePath != "" {
		file, err := os.OpenFile(fileStoragePath, os.O_RDONLY, 0777)

		if err == nil {
			defer file.Close()
			err = json.NewDecoder(file).Decode(&s.db)
		} else {
			fmt.Println("Storage could not be created from file", fileStoragePath, err)
		}

		if err == nil {
			for shortURL, originalURL := range s.db {
				s.keysDB[originalURL] = shortURL
			}
		} else {
			fmt.Println("Storage could not be created from file", fileStoragePath, err)
		}
	}

	return s
}

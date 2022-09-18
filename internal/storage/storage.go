package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

type UserUrls struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Interface interface {
	ShortenURL(originalURL string, sign string) string
	GetOriginalURL(string) (string, bool)
	GetUserArchive(sign string) ([]UserUrls, bool)
}

type Storage struct {
	Interface

	db     map[string]string
	keysDB map[string]string
	dbMux  sync.RWMutex

	usersArchive map[string][]string
	usersArcMux  sync.RWMutex

	baseURL         string
	fileStoragePath string
}

func (s *Storage) ShortenURL(originalURL string, sign string) string {
	log.Println("QWE ShortenURL", sign)

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

		if sign != "" {
			s.usersArcMux.Lock()
			s.usersArchive[sign] = append(s.usersArchive[sign], shortURL)
			s.usersArcMux.Unlock()
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

func (s *Storage) GetUserArchive(sign string) ([]UserUrls, bool) {
	log.Println("QWE GetUserArchive", s.usersArchive)

	s.usersArcMux.RLock()
	defer s.usersArcMux.RUnlock()

	urls, ok := s.usersArchive[sign]
	if !ok {
		return nil, false
	}

	res := make([]UserUrls, len(urls))

	s.dbMux.RLock()
	defer s.dbMux.RUnlock()

	for i, shortURL := range urls {
		res[i] = UserUrls{
			ShortURL:    s.baseURL + "/" + shortURL,
			OriginalURL: s.db[shortURL],
		}
	}

	return res, true
}

func Init(baseURL, fileStoragePath string) *Storage {
	s := &Storage{
		db:              make(map[string]string),
		keysDB:          make(map[string]string),
		usersArchive:    make(map[string][]string),
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

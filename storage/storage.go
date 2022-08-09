package storage

import (
	"strconv"
	"sync"
)

type Storage struct {
	db     map[string]string
	keysDB map[string]string
	dbMux  sync.Mutex
}

func (s *Storage) ShortenURL(originalURL string) string {
	s.dbMux.Lock()

	shortURL, isAlreadySaved := s.keysDB[originalURL]

	if !isAlreadySaved {
		shortURL = strconv.Itoa(len(s.db) + 1)

		s.keysDB[originalURL] = shortURL
		s.db[shortURL] = originalURL
	}

	s.dbMux.Unlock()

	return shortURL
}

func (s *Storage) GetOriginalURL(shortURL string) string {
	s.dbMux.Lock()
	originalURL := s.db[shortURL]
	s.dbMux.Unlock()

	return originalURL
}

func InitStorage() *Storage {
	return &Storage{db: make(map[string]string), keysDB: make(map[string]string), dbMux: sync.Mutex{}}
}

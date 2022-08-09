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

func (s *Storage) ShortenUrling(originalUrl string) string {
	s.dbMux.Lock()

	shortUrl, isAlreadySaved := s.keysDB[originalUrl]

	if !isAlreadySaved {
		shortUrl = strconv.Itoa(len(s.db) + 1)

		s.keysDB[originalUrl] = shortUrl
		s.db[shortUrl] = originalUrl
	}

	s.dbMux.Unlock()

	return shortUrl
}

func (s *Storage) GetOriginalUrl(shortUrl string) string {
	s.dbMux.Lock()
	originalUrl := s.db[shortUrl]
	s.dbMux.Unlock()

	return originalUrl
}

func InitStorage() *Storage {
	return &Storage{db: make(map[string]string), keysDB: make(map[string]string), dbMux: sync.Mutex{}}
}

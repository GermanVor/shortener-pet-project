package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/GermanVor/shortener-pet-project/internal/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type UserUrls struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type MappingItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type Interface interface {
	ShortenURL(originalURL string, userUUID string) (string, error)
	GetOriginalURL(shortURLId string, userUUID string) (string, error)
	GetUserArchive(userUUID string) ([]UserUrls, error)
	ForEach(mapItem []MappingItem, userUUID string, handler func(correlationID string, shortURLId string) error) error
	DeleteKeys(items []string, userUUID string) error
}

var ErrValueNotFound = errors.New("value not found")
var ErrValueGone = errors.New("value is gone")
var ErrValueAlreadyShorted = errors.New("value not found")

type setStringType map[string]bool

type V1 struct {
	Interface

	db     map[string]string
	keysDB map[string]string
	dbMux  sync.RWMutex

	usersArchive map[string]setStringType
	usersArcMux  sync.RWMutex

	baseURL         string
	fileStoragePath string
}

func (s *V1) ShortenURL(originalURL string, userUUID string) (string, error) {
	s.dbMux.Lock()

	var alreadyShortedURLErr error
	shortenURLId, isAlreadySaved := s.keysDB[originalURL]

	if !isAlreadySaved {
		shortenURLId = strconv.Itoa(len(s.db) + 1)

		s.keysDB[originalURL] = shortenURLId
		s.db[shortenURLId] = originalURL

		if s.fileStoragePath != "" {
			backupBytes, _ := json.Marshal(&s.db)
			os.WriteFile(s.fileStoragePath, backupBytes, 0644)
		}
	} else {
		alreadyShortedURLErr = ErrValueAlreadyShorted
	}

	s.dbMux.Unlock()

	if userUUID != "" {
		s.usersArcMux.Lock()
		defer s.usersArcMux.Unlock()

		if s.usersArchive[userUUID] == nil {
			s.usersArchive[userUUID] = make(setStringType)
		}

		s.usersArchive[userUUID][shortenURLId] = true
	}

	return s.baseURL + "/" + shortenURLId, alreadyShortedURLErr
}

func (s *V1) GetOriginalURL(shortenURLId string, userUUID string) (string, error) {
	if userUUID != "" {
		s.usersArcMux.RLock()
		defer s.usersArcMux.RUnlock()

		if s.usersArchive[userUUID] == nil || !s.usersArchive[userUUID][shortenURLId] {
			return "", ErrValueGone
		}
	}

	s.dbMux.RLock()
	defer s.dbMux.RUnlock()

	originalURL, ok := s.db[shortenURLId]

	if ok {
		return originalURL, nil
	}

	return "", ErrValueNotFound
}

func (s *V1) GetUserArchive(userUUID string) ([]UserUrls, error) {
	s.usersArcMux.RLock()
	defer s.usersArcMux.RUnlock()

	urls, ok := s.usersArchive[userUUID]
	if !ok {
		return nil, ErrValueNotFound
	}

	res := make([]UserUrls, len(urls))

	s.dbMux.RLock()
	defer s.dbMux.RUnlock()

	i := 0
	for shortenURLId := range urls {
		res[i] = UserUrls{
			ShortURL:    s.baseURL + "/" + shortenURLId,
			OriginalURL: s.db[shortenURLId],
		}

		i++
	}

	return res, nil
}

func (s *V1) ForEach(mapItem []MappingItem, userUUID string, handler func(CorrelationID string, ShortURL string) error) error {
	for _, iterItem := range mapItem {
		shortURL, err := s.ShortenURL(iterItem.OriginalURL, userUUID)
		if err == nil {
			err = handler(iterItem.CorrelationID, shortURL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *V1) DeleteKeys(items []string, userUUID string) error {
	s.usersArcMux.Lock()
	defer s.usersArcMux.Unlock()

	if s.usersArchive[userUUID] == nil {
		return nil
	}

	for _, shortURL := range items {
		s.usersArchive[userUUID][shortURL] = false
	}

	return nil
}

func InitV1(baseURL, fileStoragePath string) *V1 {
	s := &V1{
		db:              make(map[string]string),
		keysDB:          make(map[string]string),
		usersArchive:    make(map[string]setStringType),
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
			for shortenURLId, originalURL := range s.db {
				s.keysDB[originalURL] = shortenURLId
			}
		} else {
			fmt.Println("Storage could not be created from file", fileStoragePath, err)
		}
	}

	return s
}

type V2 struct {
	Interface

	baseURL string
	dbPool  *pgxpool.Pool
}

func InitV2(baseURL string, dbContext context.Context, connString string) (*V2, error) {
	conn, err := pgxpool.Connect(dbContext, connString)
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to DB %s successfully\n", connString)

	tx, err := conn.Begin(dbContext)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.TODO())

	{
		sql := "CREATE TABLE IF NOT EXISTS shortensArchive (" +
			"originalURL text UNIQUE, " +
			"shortenURLId SERIAL " +
			");"
		_, err = tx.Exec(context.TODO(), sql)
		if err != nil {
			return nil, err
		}
	}
	{
		sql := "CREATE TABLE IF NOT EXISTS usersArchive (" +
			"userUUID text, " +
			"shortenURLId text, " +
			"isPresent boolean DEFAULT TRUE, " +
			"PRIMARY KEY (userUUID, shortenURLId) " +
			");"
		_, err = tx.Exec(context.TODO(), sql)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit(dbContext)
	if err != nil {
		return nil, err
	}

	log.Println("Created metrics Table successfully")

	return &V2{baseURL: baseURL, dbPool: conn}, nil
}

func (s *V2) Ping() error {
	return s.dbPool.Ping(context.TODO())
}

func (s *V2) ShortenURL(originalURL string, userUUID string) (string, error) {
	shortenURLId := ""
	id := 0

	tx, _ := s.dbPool.Begin(context.TODO())
	defer tx.Rollback(context.TODO())

	var alreadyShortedURLErr error

	sql := "SELECT shortenURLId FROM shortensArchive WHERE originalURL=$1"
	err := s.dbPool.QueryRow(context.TODO(), sql, originalURL).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			sql = "INSERT INTO shortensArchive (originalURL) " +
				"VALUES ($1) ON CONFLICT (originalURL) " +
				"DO UPDATE SET originalURL=EXCLUDED.originalURL " +
				"RETURNING shortenURLId;"
			err := tx.QueryRow(context.TODO(), sql, originalURL).Scan(&id)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	} else {
		alreadyShortedURLErr = ErrValueAlreadyShorted
	}

	shortenURLId = strconv.Itoa(id)

	if userUUID != "" {
		sql = "INSERT INTO usersArchive (userUUID, shortenURLId) " +
			"VALUES ($1, $2) ON CONFLICT DO NOTHING;"
		_, err = tx.Exec(context.TODO(), sql, userUUID, shortenURLId)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(context.TODO()); err != nil {
		return "", err
	}

	return s.baseURL + "/" + shortenURLId, alreadyShortedURLErr
}

func (s *V2) GetOriginalURL(shortenURLId string, userUUID string) (string, error) {
	if userUUID != "" {
		isPresent := false
		sql := "SELECT isPresent FROM usersArchive WHERE userUUID=$1 AND shortenURLId=$2;"
		err := s.dbPool.QueryRow(context.TODO(), sql, userUUID, shortenURLId).Scan(&isPresent)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", ErrValueNotFound
			} else {
				return "", err
			}
		}

		if !isPresent {
			return "", ErrValueGone
		}
	}

	originalURL := ""

	sql := "SELECT originalURL FROM shortensArchive WHERE shortenURLId=$1"
	err := s.dbPool.QueryRow(context.TODO(), sql, shortenURLId).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrValueNotFound
		} else {
			return "", err
		}
	}

	return originalURL, nil
}

func (s *V2) GetUserArchive(userUUID string) ([]UserUrls, error) {
	sql := "SELECT shortenURLId FROM usersArchive WHERE userUUID=$1;"
	rows, _ := s.dbPool.Query(context.TODO(), sql)

	res := make([]UserUrls, 0)
	for rows.Next() {
		shortenURLId := ""
		err := rows.Scan(&shortenURLId)
		if err != nil {
			return nil, err
		}

		originalURL, err := s.GetOriginalURL(shortenURLId, userUUID)
		if err != nil {
			if errors.Is(err, ErrValueNotFound) {
				continue
			} else {
				return nil, err
			}
		}

		res = append(res, UserUrls{
			OriginalURL: originalURL,
			ShortURL:    s.baseURL + "/" + shortenURLId,
		})
	}

	return res, nil
}

func (s *V2) ForEach(mapItem []MappingItem, userUUID string, handler func(CorrelationID string, ShortURL string) error) error {
	for _, iterItem := range mapItem {
		//TODO may be better use SendBatch
		shortURL, err := s.ShortenURL(iterItem.OriginalURL, userUUID)
		if err == nil {
			err = handler(iterItem.CorrelationID, shortURL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *V2) DeleteKeys(items []string, userUUID string) error {
	tx, err := s.dbPool.Begin(context.TODO())
	if err != nil {
		return err
	}

	defer tx.Rollback(context.TODO())

	var wg sync.WaitGroup

	sql := "UPDATE usersArchive " +
		"SET isPresent=FALSE " +
		"WHERE userUUID=$1 AND shortenURLId=$2;"

	for _, shortURLsChunks := range common.Chunks(items, 15) {
		wg.Add(1)

		go func(shortURLsChunks []string) {
			defer wg.Done()

			b := &pgx.Batch{}

			for _, shortURL := range shortURLsChunks {
				b.Queue(sql, userUUID, shortURL)
			}

			batchResults := tx.SendBatch(context.TODO(), b)

			var err error
			for err == nil {
				_, err = batchResults.Exec()
				if err != nil && err.Error() != "no result" {
					tx.Rollback(context.TODO())
					return
				}
			}
		}(shortURLsChunks)
	}

	wg.Wait()

	return tx.Commit(context.TODO())
}

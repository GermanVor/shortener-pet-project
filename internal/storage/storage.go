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

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type UserUrls struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Interface interface {
	ShortenURL(originalURL string, userUUID string) (string, error)
	GetOriginalURL(string) (string, error)
	GetUserArchive(userUUID string) ([]UserUrls, error)
}

var ErrValueNotFound = errors.New("value not found")

type voidType struct{}
type setStringType map[string]voidType

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

	shortenURLId, isAlreadySaved := s.keysDB[originalURL]

	if !isAlreadySaved {
		shortenURLId = strconv.Itoa(len(s.db) + 1)

		s.keysDB[originalURL] = shortenURLId
		s.db[shortenURLId] = originalURL

		if s.fileStoragePath != "" {
			backupBytes, _ := json.Marshal(&s.db)
			os.WriteFile(s.fileStoragePath, backupBytes, 0644)
		}
	}

	s.dbMux.Unlock()

	if userUUID != "" {
		s.usersArcMux.Lock()

		if s.usersArchive[userUUID] == nil {
			s.usersArchive[userUUID] = make(setStringType)
		}

		s.usersArchive[userUUID][shortenURLId] = voidType{}

		s.usersArcMux.Unlock()
	}

	return s.baseURL + "/" + shortenURLId, nil
}

func (s *V1) GetOriginalURL(shortenURLId string) (string, error) {
	s.dbMux.RLock()
	originalURL, ok := s.db[shortenURLId]
	s.dbMux.RUnlock()

	if ok {
		return originalURL, nil
	}

	return originalURL, ErrValueNotFound
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
	defer tx.Rollback(context.TODO())

	if err != nil {
		return nil, err
	}

	{
		sql := "CREATE TABLE IF NOT EXISTS shortensArchive (" +
			"originalURL text UNIQUE, " +
			"shortenURL SERIAL " +
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

	sql := "SELECT shortenURL FROM shortensArchive WHERE originalURL=$1"
	err := s.dbPool.QueryRow(context.TODO(), sql, originalURL).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {

			sql = "INSERT INTO shortensArchive (originalURL) " +
				"VALUES ($1) ON CONFLICT (originalURL) " +
				"DO UPDATE SET originalURL=EXCLUDED.originalURL " +
				"RETURNING shortenURL;"
			err := tx.QueryRow(context.TODO(), sql, originalURL).Scan(&id)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
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

	return s.baseURL + "/" + shortenURLId, tx.Commit(context.TODO())
}

func (s *V2) GetOriginalURL(shortenURLId string) (string, error) {
	originalURL := ""

	sql := "SELECT originalURL FROM shortensArchive WHERE shortenURL=$1"
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

		originalURL, err := s.GetOriginalURL(shortenURLId)
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

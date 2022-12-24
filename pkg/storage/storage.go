package storage

import (
	"fmt"
	"sync"

	"github.com/bubu256/go-url-shortener-server/config"
)

type Storage interface {
	GetURL(string) (string, bool)
	SetNewURL(string, string) error
}

type MapDB struct {
	data sync.Map
}

func NewMapDB(cfgDB config.CfgDataBase) *MapDB {
	NewStorage := MapDB{}
	if cfgDB.InitialData != nil {
		for k, v := range cfgDB.InitialData {
			NewStorage.data.Store(k, v)
		}
	}
	return &NewStorage
}

// возвращает полный URL по ключу
func (s *MapDB) GetURL(key string) (string, bool) {

	fullURL, ok := s.data.Load(key)
	if !ok {
		return "", ok
	}

	return fullURL.(string), true
}

// сохраняет URL по ключу key в хранилище, иначе возвращает ошибку
func (s *MapDB) SetNewURL(key, URL string) error {
	if _, ok := s.data.Load(key); ok {
		err := fmt.Errorf("'%v' - уже существует в хранилище, запись не разрешена;", key)
		return err
	}

	s.data.Store(key, URL)
	return nil
}

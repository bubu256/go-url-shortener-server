package storage

import (
	"fmt"
	"sync"

	"github.com/bubu256/go-url-shortener-server/pkg/config"
)

type Storage struct {
	data sync.Map
}

func New(cfgDB config.CfgDataBase) *Storage {
	NewStorage := Storage{}
	if cfgDB.InitialData != nil {
		for k, v := range cfgDB.InitialData {
			NewStorage.data.Store(k, v)
		}
	}
	return &NewStorage
}

// возвращает полный URL по ключу
func (s *Storage) GetURL(key string) (string, bool) {

	fullURL, ok := s.data.Load(key)
	if !ok {
		//err := fmt.Errorf("'%v' отсутствует в хранилище;", key)
		// fmt.Println(err)
		return "", ok
	}

	return fullURL.(string), true
}

// сохраняет URL по ключу key в хранилище, иначе возвращает ошибку
func (s *Storage) SetNewURL(key, URL string) error {
	if _, ok := s.data.Load(key); ok {
		err := fmt.Errorf("'%v' - уже существует в хранилище, запись не разрешена;", key)
		return err
	}

	s.data.Store(key, URL)
	return nil
}

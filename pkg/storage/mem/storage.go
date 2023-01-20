package mem

import (
	"fmt"
	"sync"

	"github.com/bubu256/go-url-shortener-server/config"
	// "github.com/bubu256/go-url-shortener-server/pkg/storage"
)

// хранилище реализованное с mutex
type MapDBMutex struct {
	data  map[string]string
	mutex sync.Mutex
}

func NewMapDBMutex(cfgDB config.CfgDataBase, initData map[string]string) *MapDBMutex {
	NewStorage := MapDBMutex{}
	if initData != nil {
		for k, v := range initData {
			NewStorage.SetNewURL(k, v)
		}
	}
	return &NewStorage
}

// возвращает полный URL по ключу
func (s *MapDBMutex) GetURL(key string) (string, bool) {
	fullURL, ok := s.data[key]
	if !ok {
		return "", ok
	}

	return fullURL, true
}

// сохраняет URL по ключу key в хранилище, иначе возвращает ошибку
func (s *MapDBMutex) SetNewURL(key, URL string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.data[key]; ok {
		err := fmt.Errorf("'%v' - уже существует в хранилище, запись не разрешена;", key)
		return err
	}

	s.data[key] = URL
	return nil
}

func (s *MapDBMutex) GetLastID() (int64, bool) {
	return int64(len(s.data)), true
}

// хранилище реализованное на sync.Map
type MapDB struct {
	data sync.Map
}

func NewMapDB(cfgDB config.CfgDataBase, initData map[string]string) *MapDB {
	NewStorage := MapDB{}
	if initData != nil {
		for k, v := range initData {
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

func (s *MapDB) GetLastID() (int64, bool) {
	// считаем количество элементов
	length := int64(0)

	s.data.Range(func(_, _ interface{}) bool {
		length++
		return true
	})

	return length, true
}

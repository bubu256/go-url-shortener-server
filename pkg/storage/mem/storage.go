// Package mem implements an in-memory data store using sync.Mutex to control access to the data.
package mem

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/errorapp"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"
	"github.com/bubu256/go-url-shortener-server/pkg/helperfunc"
	"golang.org/x/exp/slices"
)

// MapDBMutex представляет собой тип, реализующий хранилище данных в памяти с использованием sync.Mutex для управления доступом к данным.
type MapDBMutex struct {
	keyToURL         map[string]string
	userToKeys       map[string][]string
	keyAvailable     map[string]bool
	connectingString string
	mutex            sync.RWMutex
}

// NewMapDBMutex - создает новый экземпляр MapDBMutex с указанными параметрами.
func NewMapDBMutex(cfgDB config.CfgDataBase, initData map[string]string) *MapDBMutex {
	// Инициализация нового хранилища в памяти с параметрами, указанными в config.CfgDataBase
	// и данными из переданной map[string]string.
	NewStorage := MapDBMutex{connectingString: cfgDB.DataBaseDSN}
	NewStorage.keyToURL = make(map[string]string)
	NewStorage.userToKeys = make(map[string][]string)
	NewStorage.keyAvailable = make(map[string]bool)
	for k, v := range initData {
		NewStorage.SetNewURL(k, v, "", true)
	}
	return &NewStorage
}

// Проверка, внутренние переменные хранилища не nil
func (s *MapDBMutex) Ping() error {
	if s.keyToURL == nil || s.userToKeys == nil {
		return errors.New("s.keyToURL == nil || s.userToKeys == nil;")
	}
	return nil
}

// SetBatchURLs - добавление пакета коротких URL-адресов в хранилище
// Возвращает список коротких ключей добавленных URL-адресов
func (s *MapDBMutex) SetBatchURLs(batch schema.APIShortenBatchInput, token string) ([]string, error) {
	result := make([]string, 0, len(batch))
	for _, elem := range batch {
		err := s.SetNewURL(elem.CorrelationID, elem.OriginalURL, token, true)
		if err != nil {
			continue
		}
		result = append(result, elem.CorrelationID)
	}
	return result, nil
}

// DeleteBatch - помечает короткие URL-адреса как недоступные при условии, что токен пользователя совпадает с создавшим URL-адрес.
func (s *MapDBMutex) DeleteBatch(chs []chan []string) error {
	for keyUser := range helperfunc.FanInSliceString(chs...) {
		s.mutex.Lock()
		if slices.Contains(s.userToKeys[keyUser[1]], keyUser[0]) {
			s.keyAvailable[keyUser[0]] = false
			s.keyToURL[keyUser[0]] = keyUser[0] + "_deleted=" + s.keyToURL[keyUser[0]]
		}
		s.mutex.Unlock()
	}
	return nil
}

// Возвращает полный URL-адрес по короткому ключу.
func (s *MapDBMutex) GetURL(key string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	fullURL, ok := s.keyToURL[key]
	if !ok {
		return "", errors.New("short key missing in mem storage;")
	}
	if !s.keyAvailable[key] {
		return "", errorapp.ErrorPageNotAvailable
	}

	return fullURL, nil
}

// GetAllURLs - возвращает все записи URL, которые были сохранены пользователем с указанным идентификатором.
// Ключи URL сохранены в виде ключей словаря, значения - в виде URL.
func (s *MapDBMutex) GetAllURLs(userID string) map[string]string {
	result := make(map[string]string)
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	keys, ok := s.userToKeys[userID]
	if !ok {
		return result
	}
	for _, k := range keys {
		fullURL, ok := s.keyToURL[k]
		if ok && s.keyAvailable[k] {
			result[k] = fullURL
		}
	}
	return result
}

// SetNewURL - сохраняет URL по ключу key в хранилище.
// Если URL уже существует в хранилище, возвращает ошибку.
func (s *MapDBMutex) SetNewURL(key, URL, tokenID string, available bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// проверяем существует ли урл
	// наверное это очень дорогая операция для проверки на дупликацию урл, но как лучше пока не знаю
	for existKey, fullURL := range s.keyToURL {
		if fullURL == URL {
			return errorapp.NewURLDuplicateError(
				fmt.Errorf("запись URL %s невозможна т.к. он уже есть базе;", URL),
				existKey,
				fullURL,
			)
		}
	}
	s.keyToURL[key] = URL
	s.userToKeys[tokenID] = append(s.userToKeys[tokenID], key)
	s.keyAvailable[key] = available
	return nil
}

// GetLastID - возвращает количество сохраненных URL в хранилище.
// Второе значение всегда true, чтобы соответствовать типу возврата других методов.
func (s *MapDBMutex) GetLastID() (int64, bool) {
	return int64(len(s.keyToURL)), true
}

// GetStats - возвращает статистику по записям из хранилища
func (s *MapDBMutex) GetStats() (schema.APIInternalStats, error) {
	return schema.APIInternalStats{URLs: len(s.keyToURL), Users: len(s.userToKeys)}, nil
}

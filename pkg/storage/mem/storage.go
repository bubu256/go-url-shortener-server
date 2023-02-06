package mem

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/errorapp"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"
)

// хранилище реализованное с mutex
type MapDBMutex struct {
	keyToURL         map[string]string
	userToKeys       map[string][]string
	connectingString string
	mutex            sync.Mutex
}

func NewMapDBMutex(cfgDB config.CfgDataBase, initData map[string]string) *MapDBMutex {
	NewStorage := MapDBMutex{connectingString: cfgDB.DataBaseDSN}
	NewStorage.keyToURL = make(map[string]string)
	NewStorage.userToKeys = make(map[string][]string)
	for k, v := range initData {
		NewStorage.SetNewURL(k, v, "")
	}
	return &NewStorage
}

func (s *MapDBMutex) Ping() error {
	if s.keyToURL == nil || s.userToKeys == nil {
		return errors.New("s.keyToURL == nil || s.userToKeys == nil;")
	}
	return nil
}

func (s *MapDBMutex) SetBatchURLs(batch schema.APIShortenBatchInput, token string) ([]string, error) {
	result := make([]string, 0, len(batch))
	for _, elem := range batch {
		err := s.SetNewURL(elem.CorrelationID, elem.OriginalURL, token)
		if err != nil {
			continue
		}
		result = append(result, elem.CorrelationID)
	}
	return result, nil
}

// возвращает полный URL по ключу
func (s *MapDBMutex) GetURL(key string) (string, bool) {
	fullURL, ok := s.keyToURL[key]
	if !ok {
		return "", ok
	}

	return fullURL, true
}

func (s *MapDBMutex) GetAllURLs(userID string) map[string]string {
	result := make(map[string]string)
	keys, ok := s.userToKeys[userID]
	if !ok {
		return result
	}
	for _, k := range keys {
		fullURL, ok := s.keyToURL[k]
		if ok {
			result[k] = fullURL
		}
	}
	return result
}

// сохраняет URL по ключу key в хранилище, иначе возвращает ошибку
func (s *MapDBMutex) SetNewURL(key, URL, tokenID string) error {
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.keyToURL[key] = URL
	s.userToKeys[tokenID] = append(s.userToKeys[tokenID], key)
	return nil
}

func (s *MapDBMutex) GetLastID() (int64, bool) {
	return int64(len(s.keyToURL)), true
}

// хранилище реализованное на sync.Map
// type MapDB struct {
// 	data sync.Map
// }

// func NewMapDB(cfgDB config.CfgDataBase, initData map[string]string) *MapDB {
// 	NewStorage := MapDB{}
// 	for k, v := range initData {
// 		NewStorage.data.Store(k, v)
// 	}

// 	return &NewStorage
// }

// // возвращает полный URL по ключу
// func (s *MapDB) GetURL(key string) (string, bool) {
// 	fullURL, ok := s.data.Load(key)
// 	if !ok {
// 		return "", ok
// 	}
// 	return fullURL.(string), true
// }

// // сохраняет URL по ключу key в хранилище, иначе возвращает ошибку
// func (s *MapDB) SetNewURL(key, URL string) error {
// 	if _, ok := s.data.Load(key); ok {
// 		err := fmt.Errorf("'%v' - уже существует в хранилище, запись не разрешена;", key)
// 		return err
// 	}

// 	s.data.Store(key, URL)
// 	return nil
// }

// func (s *MapDB) GetLastID() (int64, bool) {
// 	// считаем количество элементов
// 	length := int64(0)

// 	s.data.Range(func(_, _ interface{}) bool {
// 		length++
// 		return true
// 	})

// 	return length, true
// }

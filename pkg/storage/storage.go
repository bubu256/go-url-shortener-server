package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/bubu256/go-url-shortener-server/config"
)

type Storage interface {
	GetURL(key string) (string, bool)
	SetNewURL(key string, URL string) error
	GetLastID() (int64, bool)
}

// хранилище реализованное с mutex
type MapDBMutex struct {
	data  map[string]string
	mutex sync.Mutex
}

func NewMapDBMutex(cfgDB config.CfgDataBase) Storage {
	NewStorage := MapDBMutex{}
	if cfgDB.InitialData != nil {
		for k, v := range cfgDB.InitialData {
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
// с сохранением данных в файл
type MapDB struct {
	data sync.Map
	file *RWFile
}

func NewMapDB(cfgDB config.CfgDataBase) Storage {
	NewStorage := MapDB{}
	if cfgDB.InitialData != nil {
		for k, v := range cfgDB.InitialData {
			NewStorage.data.Store(k, v)
		}
	}
	if cfgDB.FileStoragePath == "" {
		return &NewStorage
	}
	// загружаем данные из файла если он есть
	file, err := NewRWFile(cfgDB)
	if err == nil {
		defer file.Close()
		NewStorage.file = file
		match, err := NewStorage.file.ReadMatch()
		countRead := 0
		for err == nil {
			countRead++
			NewStorage.data.Store(match.ShortKey, match.FullURL)
			match, err = NewStorage.file.ReadMatch()
		}
		log.Println("Из файла", NewStorage.file.path, "загружено элементов:", countRead)
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

	// если file существует записываем в него
	if s.file != nil {
		err := s.file.OpenAppend()
		if err != nil {
			return err
		}
		defer s.file.Close()
		s.file.WriteMatch(Match{ShortKey: key, FullURL: URL})
	}
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

// структура для сериализации данных
type Match struct {
	ShortKey string `json:"short_key"`
	FullURL  string `json:"full_url"`
}

// структура для работы с файлом
type RWFile struct {
	path    string
	file    *os.File
	encoder *json.Encoder
	decoder *json.Decoder
}

// создает структуру с открытым файлом на чтение/запись и decoder, encoder
func NewRWFile(cfgDB config.CfgDataBase) (*RWFile, error) {
	file, err := os.OpenFile(cfgDB.FileStoragePath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Println("Не удалось открыть файл;", err, "; path", cfgDB.FileStoragePath)
		return nil, err
	}
	rwf := RWFile{file: file, encoder: json.NewEncoder(file), decoder: json.NewDecoder(file), path: cfgDB.FileStoragePath}
	return &rwf, nil
}

func (r *RWFile) WriteMatch(match Match) error {
	return r.encoder.Encode(match)
}

func (r *RWFile) ReadMatch() (*Match, error) {
	match := Match{}
	err := r.decoder.Decode(&match)
	if err != nil {
		return nil, err
	}
	return &match, nil
}

func (r *RWFile) Close() error {
	r.decoder = nil
	r.encoder = nil
	return r.file.Close()
}

// открывает файл для записи в конец и создает encoder
func (r *RWFile) OpenAppend() error {
	file, err := os.OpenFile(r.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		log.Println("Не удалось открыть файл для записи;", err)
		return err
	}
	r.file = file
	r.encoder = json.NewEncoder(file)
	return nil
}

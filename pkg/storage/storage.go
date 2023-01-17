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

func New(cfgDB config.CfgDataBase) Storage {
	// создаем базовый Storage
	newStorage := NewMapDB(cfgDB)
	// если указан путь к файлу создаем Storage с чтением/записью в файл
	if cfgDB.FileStoragePath != "" {
		fileStorage, err := NewWrapToSaveFile(cfgDB.FileStoragePath, newStorage)
		if err == nil {
			return fileStorage
		}
	}
	return newStorage
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
type MapDB struct {
	data sync.Map
}

func NewMapDB(cfgDB config.CfgDataBase) Storage {
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

func (s *MapDB) GetLastID() (int64, bool) {
	// считаем количество элементов
	length := int64(0)
	s.data.Range(func(_, _ interface{}) bool {
		length++
		return true
	})

	return length, true
}

// структура для Storage дополнительно сохраняет данные в файл
type WrapToSaveFile struct {
	storage Storage
	file    *RWFile
}

func (s *WrapToSaveFile) SetNewURL(key string, URL string) error {
	// пишем в файл и вызываем стандартный обработчик
	err := s.file.OpenAppend()
	if err != nil {
		return err
	}
	defer s.file.Close()
	s.file.WriteMatch(Match{ShortKey: key, FullURL: URL})

	return s.storage.SetNewURL(key, URL)
}

func (s *WrapToSaveFile) GetURL(key string) (string, bool) {
	return s.storage.GetURL(key)
}

func (s *WrapToSaveFile) GetLastID() (int64, bool) {
	return s.storage.GetLastID()
}

// Возвращает Storage с на основе исходного (st) с возможность работать с файлом
func NewWrapToSaveFile(pathFile string, st Storage) (Storage, error) {
	//загружаем данные из файла если он есть
	file, err := NewRWFile(pathFile)
	if err != nil {
		return st, err
	}
	defer file.Close()
	file.path = pathFile
	match, err := file.ReadMatch()
	countRead := 0
	for err == nil {
		countRead++
		st.SetNewURL(match.ShortKey, match.FullURL)
		match, err = file.ReadMatch()
	}
	log.Println("Из файла", file.path, "загружено элементов:", countRead)

	return &WrapToSaveFile{storage: st, file: file}, nil
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
func NewRWFile(pathFile string) (*RWFile, error) {
	file, err := os.OpenFile(pathFile, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Println("Не удалось открыть файл;", err, "; path", pathFile)
		return nil, err
	}
	rwf := RWFile{file: file, encoder: json.NewEncoder(file), decoder: json.NewDecoder(file), path: pathFile}
	return &rwf, nil
}

func (r *RWFile) WriteMatch(match Match) error {
	return r.encoder.Encode(match)
}

// декодирует элемент Match из файла
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

package storage

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"
	"github.com/bubu256/go-url-shortener-server/pkg/storage/mem"
	"github.com/bubu256/go-url-shortener-server/pkg/storage/postgres"
)

type Storage interface {
	GetURL(key string) (string, error)
	GetAllURLs(userID string) map[string]string
	SetNewURL(key, URL, tokenID string, available bool) error
	GetLastID() (int64, bool)
	Ping() error
	SetBatchURLs(batch schema.APIShortenBatchInput, token string) ([]string, error)
}

func New(cfgDB config.CfgDataBase, initData map[string]string) Storage {
	if cfgDB.DataBaseDSN != "" {
		db, err := postgres.New(cfgDB)
		if err == nil {
			return db
		}
		log.Println(err)
	}
	// создаем базовый Storage mem
	newStorage := mem.NewMapDBMutex(cfgDB, initData)
	// log.Printf("%v", newStorage)
	// если указан путь к файлу создаем Storage с чтением/записью в файл
	if cfgDB.FileStoragePath != "" {
		fileStorage, err := NewWrapToSaveFile(cfgDB.FileStoragePath, newStorage)
		if err == nil {
			return fileStorage
		}
	}
	return newStorage
}

// структура для Storage дополнительно сохраняет данные в файл
type WrapToSaveFile struct {
	storage Storage
	file    *RWFile
}

func (s *WrapToSaveFile) SetNewURL(key, URL, TokenID string, available bool) error {
	// пишем в файл и вызываем стандартный обработчик
	err := s.file.OpenAppend()
	if err != nil {
		return err
	}
	defer s.file.Close()
	s.file.WriteMatch(Match{ShortKey: key, FullURL: URL, UserID: TokenID, Available: &available})

	return s.storage.SetNewURL(key, URL, TokenID, available)
}

func (s *WrapToSaveFile) GetURL(key string) (string, error) {
	return s.storage.GetURL(key)
}

func (s *WrapToSaveFile) GetLastID() (int64, bool) {
	return s.storage.GetLastID()
}

func (s *WrapToSaveFile) GetAllURLs(userID string) map[string]string {
	return s.storage.GetAllURLs(userID)
}

func (s *WrapToSaveFile) Ping() error {
	return s.storage.Ping()
}

func (s *WrapToSaveFile) SetBatchURLs(batch schema.APIShortenBatchInput, token string) ([]string, error) {
	return s.storage.SetBatchURLs(batch, token)
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
	countRead := 0
	match, err := file.ReadMatch()
	for err == nil {
		countRead++
		if match.Available == nil {
			return nil, errors.New("match.Available == nil, хотя должен быть true od false")
		}
		st.SetNewURL(match.ShortKey, match.FullURL, match.UserID, *(match.Available))
		match, err = file.ReadMatch()
	}
	if err != io.EOF {
		return nil, err
	}

	log.Println("Из файла", file.path, "загружено элементов:", countRead)
	return &WrapToSaveFile{storage: st, file: file}, nil
}

// структура для сериализации данных
// Match.Available *bool я использую тут указатель что бы можно было отследить
// отсутствие поля и установить значение по умолчанию true
type Match struct {
	ShortKey  string `json:"short_key"`
	FullURL   string `json:"full_url"`
	UserID    string `json:"user_id"`
	Available *bool  `json:"available"` // default true
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
	if match.Available == nil {
		defaultTrue := true
		match.Available = &defaultTrue
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

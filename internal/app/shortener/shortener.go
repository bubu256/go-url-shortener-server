package shortener

import (
	"strconv"

	"github.com/bubu256/go-url-shortener-server/pkg/storage"
)

// lastID - уникальный ключ. Так как хранилище реализовано с потокобезопасной структурой,
// тут решено использовать канал во избежании создания одинаковых ID разными горутинами (если они будут конечно)
// не уверен что надо делать так - но пока это работает
type Shortener struct {
	db     *storage.Storage
	lastID chan int
}

func New(db *storage.Storage) *Shortener {
	NewSh := Shortener{db: db}
	NewSh.lastID = make(chan int, 1)
	NewSh.lastID <- 100
	return &NewSh
}

// создает и возвращает новый ключ
func (s *Shortener) getNewKey() string {
	var current int = <-s.lastID
	current++
	s.lastID <- current
	return strconv.Itoa(current)
}

// возвращает короткий ключ; полный URL сохраняет в хранилище
func (s *Shortener) CreateShortURL(fullURL string) (shortKey string, err error) {
	key := s.getNewKey()
	err = s.db.SetNewURL(key, fullURL)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *Shortener) GetURL(shortKey string) (string, bool) {
	return s.db.GetURL(shortKey)
}

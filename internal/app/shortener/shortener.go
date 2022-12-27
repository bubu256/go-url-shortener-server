package shortener

import (
	"strconv"
	"sync/atomic"

	"github.com/bubu256/go-url-shortener-server/pkg/storage"
)

type Shortener struct {
	// возможно лучше использовать ссылку на интерфейс? *storage.Storage
	db     storage.Storage
	lastID atomic.Int64
}

func New(db storage.Storage) *Shortener {
	NewSh := Shortener{db: db}
	// инициализации lastID под вопросом, по идеи его нужно загружать из БД по последней записи
	// или переписать логику создания ключа на рандомные значения и последующей проверки на уникальность
	// но с этим как я понял получится определиться и позже, когда будет внедрение БД
	NewSh.lastID.Store(100)
	return &NewSh
}

// создает и возвращает новый ключ
func (s *Shortener) getNewKey() string {
	return strconv.Itoa(int(s.lastID.Add(1)))
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

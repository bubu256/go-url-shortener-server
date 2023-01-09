package shortener

import (
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/bubu256/go-url-shortener-server/pkg/storage"
)

const (
	// символы для короткого ключа
	basicSymbols = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	baseKey      = len(basicSymbols)
)

type Shortener struct {
	db     storage.Storage
	lastID atomic.Int64
	// количество случайных символов в конце ключа
	rndSymbolsEnd int
}

func New(db storage.Storage) *Shortener {
	rand.Seed(time.Now().Unix())

	NewSh := Shortener{
		db:            db,
		rndSymbolsEnd: 3,
	}
	lastID, ok := db.GetLastID()
	if ok {
		NewSh.lastID.Store(lastID)
	} else {
		log.Println("не удалось получить последний id из хранилища. LastID установлен 100000")
		NewSh.lastID.Store(100)
	}
	return &NewSh
}

// создает и возвращает новый ключ состоящий из закодированного id и случайных символов в конце
func (s *Shortener) getNewKey() string {
	// инкриминируем и получаем id
	id := int(s.lastID.Add(1))
	if id < 0 {
		id = -id
	}
	baseSize := 6
	codeByte := make([]byte, baseSize)
	// кодируем id в базовые символы. заполняем слайс с конца.
	i := 0
	for res := id; res > 0; res /= baseKey {
		i++
		index := res % baseKey
		// если не хватило места расширяем слайс
		if i > baseSize {
			baseSize *= 2
			codeByte = append(codeByte, codeByte...)
		}
		codeByte[baseSize-i] = basicSymbols[index]
	}
	// добавляем в конец rndSymbolsEnd случайных символа
	for i := 0; i < s.rndSymbolsEnd; i++ {
		rnd := rand.Intn(baseKey)
		codeByte = append(codeByte, basicSymbols[rnd])
	}
	return string(codeByte[baseSize-i:])
}

// возвращает короткий ключ; полный URL сохраняет в хранилище
func (s *Shortener) CreateShortKey(fullURL string) (shortKey string, err error) {
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

package shortener

import (
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/data"
	"github.com/bubu256/go-url-shortener-server/pkg/storage"
)

const (
	// символы для короткого ключа
	basicSymbols = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	baseKey      = len(basicSymbols)
)

type Shortener struct {
	db            storage.Storage
	lastID        atomic.Int64
	rndSymbolsEnd int // количество случайных символов в конце ссылки-ключа
	secretKey     []byte
}

func New(db storage.Storage, cfg config.CfgService) *Shortener {
	rand.Seed(time.Now().Unix())

	// установка секретного ключа
	var keyByte []byte
	if cfg.SecretKey != "" {
		hexdecode, err := hex.DecodeString(cfg.SecretKey)
		if err != nil {
			log.Println("ошибка декодирования секретного ключа (hex);")
			genKey, err := GenerateRandomBytes(32)
			if err != nil {
				log.Fatal("ошибка при генерации секретного ключа (shortener new generateRandomKey);")
			}
			log.Printf("создан рандомный ключ: %x", genKey)
			keyByte = genKey
		} else {
			keyByte = hexdecode
		}
	} else {
		genKey, err := GenerateRandomBytes(32)
		if err != nil {
			log.Fatal("ошибка при генерации секретного ключа (shortener new generateRandomKey);")
		}
		log.Printf("создан рандомный ключ: %x", genKey)
		keyByte = genKey
	}
	// создание сервиса
	NewSh := Shortener{
		db:            db,
		rndSymbolsEnd: 3,
		secretKey:     keyByte,
	}
	// инициализация счетчика количества записей
	lastID, ok := db.GetLastID()
	if ok {
		NewSh.lastID.Store(lastID)
	} else {
		log.Println("не удалось получить последний id из хранилища. LastID установлен 100000")
		NewSh.lastID.Store(100000)
	}
	return &NewSh
}

func (s *Shortener) SetBatchURLs(batch data.APIShortenBatchInput, token string) ([]string, error) {
	return s.db.SetBatchURLs(batch, token)
}

// пингует ДБ
func (s *Shortener) PingDB() error {
	return s.db.Ping()
}

// генерирует рандомный набор байт
func GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := crand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// генерирует новый токен для пользователя
func (s *Shortener) GenerateNewToken() (string, error) {
	idUser, err := GenerateRandomBytes(4)
	if err != nil {
		return "", err
	}
	h := hmac.New(sha256.New, s.secretKey)
	h.Write(idUser)
	dst := h.Sum(nil)
	dst = append(idUser, dst...) // содержит байты id и подписи
	// кодируем в hex и отдаем как токен в виде строки
	return hex.EncodeToString(dst), nil
}

// проверяет подлинность токена
func (s *Shortener) CheckToken(token string) bool {
	decodeToken, err := hex.DecodeString(token)
	if err != nil {
		return false
	}
	idUser := decodeToken[:4]
	sing := decodeToken[4:]
	h := hmac.New(sha256.New, s.secretKey)
	h.Write(idUser)
	dst := h.Sum(nil)
	return hmac.Equal(sing, dst)
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
func (s *Shortener) CreateShortKey(fullURL, tokenID string) (shortKey string, err error) {
	key := s.getNewKey()
	err = s.db.SetNewURL(key, fullURL, tokenID)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *Shortener) GetURL(shortKey string) (string, bool) {
	return s.db.GetURL(shortKey)
}

func (s *Shortener) GetAllURLs(tokenID string) map[string]string {
	// result := make(map[string]string)
	return s.db.GetAllURLs(tokenID)
}

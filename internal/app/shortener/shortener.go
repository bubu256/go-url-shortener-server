// Пакет предоставляет структуру для работы сервиса сокращения ссылок.
package shortener

import (
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"
	"github.com/bubu256/go-url-shortener-server/pkg/storage"
)

// константы участвующие в создании короткой ссылки
const (
	// символы для короткого ключа
	basicSymbols = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	baseKey      = len(basicSymbols)
)

// CounterID - структура хранящая последний выданный ID короткого идентификатора.
// При выдаче след ID икрементирует свою внутреннюю переменную.
// Для правильной работы со структурой необходимо после инициализации вызывать метод Run() для запуска инкрементирующей горутины.
type CounterID struct {
	lastID int
	output chan int
}

// NewCounter - функция, которая создает ссылку на экземпляр счетчика
func NewCounter(lastID int) *CounterID {
	return &CounterID{
		lastID: lastID,
		output: make(chan int),
	}
}

// increment - метод, который увеличивает счетчик на единицу и отправляет его значение в канал.
func (c *CounterID) increment() {
	c.lastID++
	c.output <- c.lastID
}

// Next - метод, который возвращает значение счетчика
func (c *CounterID) Next() int {
	return <-c.output
}

// Run - внутри запускает горутину инкрементирующую счетчик при вызове метода Next()
func (c *CounterID) Run() {
	go func() {
		for {
			c.increment()
		}
	}()
}

// Shortener представляет собой объект, отвечающий за генерацию и хранение коротких ссылок
type Shortener struct {
	db            storage.Storage
	lastID        *CounterID
	rndSymbolsEnd int // количество случайных символов в конце ссылки-ключа
	secretKey     []byte
}

// New создает ссылку на новый объект Shortener с переданными параметрами
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
		// NewSh.lastID.Store(lastID)
		NewSh.lastID = NewCounter(int(lastID))
	} else {
		log.Println("не удалось получить последний id из хранилища. LastID установлен 100000")
		// NewSh.lastID.Store(100000)
		NewSh.lastID = NewCounter(100000)
	}
	NewSh.lastID.Run()
	return &NewSh
}

// SetBatchURLs - осуществляет пакетную установку множества ссылок в хранилище.
// Функция принимает входные данные batch типа schema.APIShortenBatchInput и token типа string,
// и возвращает слайс строк с короткими идентификаторами ссылок и ошибку типа error.
func (s *Shortener) SetBatchURLs(batch schema.APIShortenBatchInput, token string) ([]string, error) {
	return s.db.SetBatchURLs(batch, token)
}

// DeleteBatch - осуществляет пакетное удаление множества ссылок из хранилища.
// Функция принимает входные данные batchShortKeys типа []string с короткими идентификаторами ссылок,
// token типа string, и не возвращает значения.
func (s *Shortener) DeleteBatch(batchShortKeys []string, token string) {
	numCh := 4
	inputChs := make([]chan []string, 0, numCh)
	for i := 0; i < numCh; i++ {
		inCh := make(chan []string)
		inputChs = append(inputChs, inCh)
	}
	go func() {
		for i, key := range batchShortKeys {
			inputChs[i%numCh] <- []string{key, token}
		}
		for _, ch := range inputChs {
			close(ch)
		}
	}()

	err := s.db.DeleteBatch(inputChs)
	if err != nil {
		log.Println(fmt.Errorf("сервис получил ошибку при удалении данных из хранилища; %w", err))
	}
}

// PingDB - пингует БД
func (s *Shortener) PingDB() error {
	return s.db.Ping()
}

// GenerateRandomBytes - генерирует рандомный набор байт
func GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := crand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateNewToken генерирует новый токен для пользователя на основе рандомного набора байт в качестве id пользователя
// и секретного ключа, используя алгоритм HMAC-SHA256.
// Возвращает токен в виде строки в шестнадцатеричном виде и ошибку, если таковая возникла.
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

// CheckToken проверяет подлинность токена, переданного в виде строки в шестнадцатеричном виде,
// используя секретный ключ и алгоритм HMAC-SHA256. Возвращает true, если токен подлинный и false, если нет.
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

// getNewKey - создает и возвращает новый ключ состоящий из закодированного id и случайных символов в конце.
// Для кодирования используется базовый алфавит, состоящий из 64 символов, а также случайные символы,
// генерируемые с помощью пакета rand.
// Функция использует счетчик последнего выданного id для генерации нового ключа.
// Возвращает сгенерированный ключ в виде строки.
func (s *Shortener) getNewKey() string {
	// инкриминируем и получаем id
	id := s.lastID.Next()
	if id < 0 {
		id = -id
	}
	// тут твориться магия по заполнению слайса с конца.
	// это работает быстрее чем заполнение слайса обычном методом и потом его разворот
	baseSize := 6
	codeByte := make([]byte, baseSize)
	// кодируем id в базовые символы. заполняем слайс с конца.
	i := 0 // используется как метка сколько байт было записано, нужна для возврата значения
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

// getNewKeyV2 создает и возвращает новый ключ, используя генерацию случайных байтов с помощью пакета rand,
// Это вторая версия метода getNewKey была написана для проведения бенчмарков и профилирования.
func (s *Shortener) getNewKeyV2() string {
	var randomBytes [6]byte
	_, err := rand.Read(randomBytes[:])
	if err != nil {
		log.Print(err)
		return ""
	}

	// Преобразуем байты в строку с использованием базового 64-ричного алфавита
	encoded := base64.RawURLEncoding.EncodeToString(randomBytes[:])
	return encoded
}

// CreateShortKey генерирует новый короткий ключ для полного URL и сохраняет его в хранилище
//
// fullURL - полный URL, для которого нужно сгенерировать короткий ключ
// tokenID - идентификатор пользователя, для которого генерируется ключ
//
// Возвращает короткий ключ, созданный для полного URL, и ошибку, если таковая произошла.
func (s *Shortener) CreateShortKey(fullURL, tokenID string) (shortKey string, err error) {
	key := s.getNewKey()
	err = s.db.SetNewURL(key, fullURL, tokenID, true)
	if err != nil {
		return "", err
	}
	return key, nil
}

// GetURL получает полный URL по заданному короткому ключу
//
// shortKey - короткий ключ, для которого нужно получить полный URL
//
// Возвращает полный URL, связанный с данным коротким ключом, и ошибку, если таковая произошла.
func (s *Shortener) GetURL(shortKey string) (string, error) {
	return s.db.GetURL(shortKey)
}

// GetAllURLs получает все URL, связанные с заданным идентификатором пользователя
//
// tokenID - идентификатор пользователя, для которого нужно получить все URL
//
// Возвращает map[string]string, где ключ - короткий ключ, а значение - соответствующий полный URL
func (s *Shortener) GetAllURLs(tokenID string) map[string]string {
	return s.db.GetAllURLs(tokenID)
}

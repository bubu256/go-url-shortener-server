package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bubu256/go-url-shortener-server/internal/app/errorapp"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	Router  http.Handler
	service *shortener.Shortener
	baseURL string
}

func New(service *shortener.Shortener, cfgServer config.CfgServer) *Handlers {
	if service == nil {
		log.Fatal("указатель на структуру shortener.Shortener должен быть != nil;")
	}
	NewHandlers := Handlers{baseURL: cfgServer.BaseURL}
	NewHandlers.service = service
	router := chi.NewRouter()
	router.Use(gzipWriter, gzipReader, NewHandlers.TokenHandler)
	router.Post("/", NewHandlers.HandlerURLtoShort)
	router.Get("/{ShortKey}", NewHandlers.HandlerShortToURL)
	router.Post("/api/shorten", NewHandlers.HandlerAPIShorten)
	router.Get("/api/user/urls", NewHandlers.HandlerAPIUserAllURLs)
	router.Delete("/api/user/urls", NewHandlers.HandlerAPIDeleteUrls)
	router.Post("/api/shorten/batch", NewHandlers.HandlerAPIShortenBatch)
	router.Get("/ping", NewHandlers.HandlerPing)
	NewHandlers.Router = router
	return &NewHandlers
}

// GET пингует базу данных
func (h *Handlers) HandlerPing(w http.ResponseWriter, r *http.Request) {
	err := h.service.PingDB()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// обработчик Post запросов, возвращает сокращенный URL в теле ответа
func (h *Handlers) HandlerURLtoShort(w http.ResponseWriter, r *http.Request) {
	StatusCode := http.StatusCreated
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Не удалось прочитать тело POST запроса.", http.StatusInternalServerError)
		return
	}
	fullURL := string(body)
	// получаем токен
	token, err := GetToken(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
	}
	// получаем короткий идентификатор ссылки
	shortKey, err := h.service.CreateShortKey(fullURL, token)
	var errDuplicate *errorapp.URLDuplicateError
	if errors.As(err, &errDuplicate) {
		// если ошибка дубликации урл
		StatusCode = http.StatusConflict
		shortKey = errDuplicate.ExistsKey
	} else if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// собираем сокращенную ссылку и пишем в тело
	shortURL, err := h.createLink(shortKey)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(StatusCode)
	w.Write([]byte(shortURL))
}

// обработчик Get запросов, возвращает полный URL в заголовке ответа Location
// router.Get "/{ShortKey}"
func (h Handlers) HandlerShortToURL(w http.ResponseWriter, r *http.Request) {
	shortKey := chi.URLParam(r, "ShortKey")
	fullURL, err := h.service.GetURL(shortKey)
	if err != nil {
		if err == errorapp.ErrorPageNotAvailable {
			w.WriteHeader(http.StatusGone)
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", fullURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// POST записывает сокращенный идентификатор и полный урл в хранилище
func (h *Handlers) HandlerAPIShortenBatch(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// читаем тело
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("не удалось прочитать body;")
		return
	}
	// парсим json
	batch := schema.APIShortenBatchInput{}
	err = json.Unmarshal(body, &batch)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("не удалось декодировать json;")
		return
	}
	// получаем токен
	token, err := GetToken(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// получаем идентификаторы ссылок записанные в базу
	shortKeys, err := h.service.SetBatchURLs(batch, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// собираем структуру для вывода
	batchOut := make(schema.APIShortenBatchOutput, len(shortKeys))
	for i, key := range shortKeys {
		batchOut[i].CorrelationID = key
		fullURL, err := h.createLink(key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		batchOut[i].ShortURL = fullURL
	}
	// пишем ответ в json формате
	result, err := json.Marshal(batchOut)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(result)
}

// POST возвращает сокращенный URL в json формате
func (h *Handlers) HandlerAPIShorten(w http.ResponseWriter, r *http.Request) {
	StatusCode := http.StatusCreated
	// читаем запрос
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Не удалось прочитать тело POST запроса.", http.StatusInternalServerError)
		return
	}
	// парсим входные данные
	inputData := schema.APIShortenInput{}
	err = json.Unmarshal(body, &inputData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// получаем токен
	token, err := GetToken(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	// получаем созданный короткий ключ для URL
	shortKey, err := h.service.CreateShortKey(inputData.URL, token)
	var errDuplicate *errorapp.URLDuplicateError
	if errors.As(err, &errDuplicate) {
		// если ошибка дубликации урл
		StatusCode = http.StatusConflict
		shortKey = errDuplicate.ExistsKey
	} else if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// собираем короткую ссылку
	shortURL, err := h.createLink(shortKey)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// пишем ответ
	output := schema.APIShortenOutput{Result: shortURL}
	result, err := json.Marshal(output)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(StatusCode)
	w.Write(result)
}

// GET возвращает пользователю все его сокращенные и полные URL в json формате
func (h *Handlers) HandlerAPIUserAllURLs(w http.ResponseWriter, r *http.Request) {
	token, err := GetToken(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
	}
	// запрашиваем мапу со всеми url пользователя
	allURLs := h.service.GetAllURLs(token)
	if len(allURLs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	outUrls := make(schema.APIUserURLs, len(allURLs))
	i := 0
	for k, v := range allURLs {
		shortURL, err := h.createLink(k)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		outUrls[i].ShortURL = shortURL
		outUrls[i].OriginalURL = v
		i++
	}

	result, err := json.Marshal(outUrls)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

// router.Delete("/api/user/urls", NewHandlers.HandlerAPIDeleteUrls)
func (h *Handlers) HandlerAPIDeleteUrls(w http.ResponseWriter, r *http.Request) {
	token, err := GetToken(r)
	if err != nil {
		log.Println(fmt.Errorf("при получении токена в HandlerAPIDeleteUrls произошла ошибка; %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	batchShortUrls := []string{}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(fmt.Errorf("в HandlerAPIDeleteUrls при чтении тела запроса произошла ошибка; %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &batchShortUrls)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.service.DeleteBatch(batchShortUrls, token)
	w.WriteHeader(http.StatusAccepted)
}

// функция принимает ключ и возвращает короткую ссылку на основе Handlers.baseURL
func (h *Handlers) createLink(shortKey string) (string, error) {
	return url.JoinPath(h.baseURL, shortKey)
}

// структура для подмены writer
type newWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w newWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Middleware функция проверяет и выдает токен в куках для аутентификации
func (h *Handlers) TokenHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("token")
		if err != nil || !h.service.CheckToken(token.Value) {
			newToken, err := h.service.GenerateNewToken()
			if err != nil {
				log.Println("ошибка при генерации токена;", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// ставим новые куки и в ответ и в запрос
			cookie := &http.Cookie{Name: "token", Value: newToken, Path: "/"}
			http.SetCookie(w, cookie)
			// token.Value = newToken // меняем в request значение токена на новый
			// r.Cookies()
			r.AddCookie(cookie) // это не работает
		}
		next.ServeHTTP(w, r)
	})
}

// Middleware функция подменяет responsewriter если требуется сжатие gzip в ответе
func gzipWriter(next http.Handler) http.Handler {
	// используем замыкание чтобы не создавать каждый раз новый объект используя NewWriterLevel
	// есть ли тут проблемы с потокобезопасностью?
	gzWriter, err := gzip.NewWriterLevel(&bytes.Buffer{}, gzip.BestSpeed)
	if err != nil {
		log.Println("ошибка при создании gzWriter:", err)
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gzWriter.Reset(w)
		defer gzWriter.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(newWriter{ResponseWriter: w, Writer: gzWriter}, r)
	})
}

// Middleware функция для POST распаковывает сжатый gzip (Content-Type: gzip)
func gzipReader(next http.Handler) http.Handler {
	// готовим буфер для последующего создания ридера
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte{})
	if err != nil {
		log.Fatal(err)
	}
	defer gw.Close()
	// создаем ридер
	gzReader, err := gzip.NewReader(&buf)
	if err != nil {
		log.Fatal(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}
		gzReader.Reset(r.Body)
		defer gzReader.Close()
		r.Body = gzReader
		next.ServeHTTP(w, r)
	})
}

// добываем последний cookie token (если токен был не верный, то был выдан новый и он последний, его и берем)
func GetToken(r *http.Request) (string, error) {
	cookies := r.Cookies()
	cookie := new(http.Cookie)
	for _, c := range cookies {
		if c.Name == "token" {
			cookie = c
		}
	}
	if cookie == nil {
		return "", errors.New("токен не найден;")
	}
	return cookie.Value, nil
}

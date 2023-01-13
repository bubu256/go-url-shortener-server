package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	Router  http.Handler
	service *shortener.Shortener
	baseURL url.URL
}

// структура для принятия данных в запросе из json
type InputData struct {
	URL string `json:"url"`
}

type OutputData struct {
	Result string `json:"result"`
}

func New(service *shortener.Shortener, cfgServer config.CfgServer) *Handlers {
	// парсим базовый url из конфига
	baseURL, err := url.Parse(cfgServer.BaseURL)
	if err != nil || baseURL.Host == "" {
		// если не вышло используем базовый url сервера и схему из конфига
		baseURL.Scheme = cfgServer.Scheme
		baseURL.Host = cfgServer.ServerAddress
	}

	NewHandlers := Handlers{baseURL: *baseURL}
	NewHandlers.service = service
	router := chi.NewRouter()
	router.Use(gzipWriter, gzipReader)
	router.Post("/", NewHandlers.HandlerURLtoShort)
	router.Post("/api/shorten", NewHandlers.HandlerAPIShorten)
	router.Get("/{ShortKey}", NewHandlers.HandlerShortToURL)
	NewHandlers.Router = router
	return &NewHandlers
}

// обработчик Post запросов, возвращает сокращенный URL в теле ответа
func (h *Handlers) HandlerURLtoShort(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Не удалось прочитать тело POST запроса.", http.StatusInternalServerError)
		return
	}
	shortKey, err := h.service.CreateShortKey(string(body))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// собираем сокращенную ссылку и пишем в тело
	shortURL, err := h.CreateLink(shortKey)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// обработчик Get запросов, возвращает полный URL в заголовке ответа Location
func (h Handlers) HandlerShortToURL(w http.ResponseWriter, r *http.Request) {
	shortKey := chi.URLParam(r, "ShortKey")
	fullURL, ok := h.service.GetURL(shortKey)
	if ok {
		w.Header().Set("Location", fullURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (h *Handlers) HandlerAPIShorten(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Не удалось прочитать тело POST запроса.", http.StatusInternalServerError)
		return
	}
	inputData := InputData{}
	err = json.Unmarshal(body, &inputData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	shortKey, err := h.service.CreateShortKey(inputData.URL)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	shortURL, err := h.CreateLink(shortKey)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outputData := OutputData{Result: shortURL}
	result, err := json.Marshal(outputData)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(result)
}

// функция принимает ключ и возвращает короткую ссылку с baseURL
func (h *Handlers) CreateLink(shortKey string) (string, error) {
	return url.JoinPath(h.baseURL.String(), shortKey)
}

// структура для подмены writer
type newWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w newWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
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

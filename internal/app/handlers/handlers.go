package handlers

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	Router  http.Handler
	service *shortener.Shortener
}

func New(service *shortener.Shortener) *Handlers {
	NewHandlers := Handlers{}
	NewHandlers.service = service
	router := chi.NewRouter()
	router.Post("/", NewHandlers.HandlerURLtoShort)
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
	shortKey, err := h.service.CreateShortURL(string(body))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// собираем сокращенную ссылку и пишем в тело
	shortURL := url.URL{
		Scheme: "http",
		Host:   r.Host,
		Path:   shortKey,
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL.String()))
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

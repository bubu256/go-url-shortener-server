package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	Router  http.Handler
	service *shortener.Shortener
}

func New(service *shortener.Shortener) *Handlers {
	router := chi.NewRouter()
	router.Post("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Не удалось прочитать тело POST запроса.", http.StatusInternalServerError)
			return
		}
		shortKey, err := service.CreateShortURL(string(body))
		if err != nil {
			log.Fatal(err)
			w.WriteHeader(500)
			return
		}
		// собираем ссылку и пишем в тело
		shortURL := "http://" + r.Host + "/" + shortKey
		w.WriteHeader(201)
		w.Write([]byte(shortURL))
	})

	router.Get("/{ShortKey}", func(w http.ResponseWriter, r *http.Request) {
		shortKey := chi.URLParam(r, "ShortKey")
		fullURL, ok := service.GetURL(shortKey)
		if ok {
			w.Header().Set("Location", fullURL)
			w.WriteHeader(307)
		} else {
			w.WriteHeader(400)
		}
	})

	NewHandlers := Handlers{}
	NewHandlers.service = service
	NewHandlers.Router = router
	return &NewHandlers
}

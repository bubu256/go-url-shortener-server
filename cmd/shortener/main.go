package main

import (
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

var baseURL map[string]string
var lastID int

func init() {
	baseURL = make(map[string]string)
}

func main() {
	router := chi.NewRouter()
	router.Post("/", HandlerURLtoShort)
	router.Get("/{ShortKey}", HandlerShortToFullURL)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func HandlerShortToFullURL(w http.ResponseWriter, r *http.Request) {
	shortKey := r.URL.Path[1:]
	fullURL, ok := baseURL[shortKey]
	if ok {
		w.Header().Set("Location", fullURL)
		w.WriteHeader(307)
	} else {
		w.WriteHeader(400)
	}
}

func HandlerURLtoShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Не удалось прочитать тело POST запроса.", http.StatusInternalServerError)
		return
	}
	lastID++
	shortKey := strconv.Itoa(lastID)
	baseURL[shortKey] = string(body)
	// создание короткой ссылки из хоста и shortKey
	shortURL := "http://" + r.Host + "/" + shortKey
	w.WriteHeader(201)
	w.Write([]byte(shortURL))
}

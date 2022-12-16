package main

import (
	"io"
	"net/http"
	"strconv"
)

var baseURL map[string]string
var lastID int

func Shortener(w http.ResponseWriter, r *http.Request) {
	// Обработка метода Post
	if r.Method == http.MethodPost {
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

	// Обработка метода Get
	if r.Method == http.MethodGet {
		shortKey := r.URL.Path[1:]
		fullURL, ok := baseURL[shortKey]
		if ok {
			w.Header().Set("Location", fullURL)
			w.WriteHeader(307)
		} else {
			w.WriteHeader(400)
		}
	}

}

func init() {
	baseURL = make(map[string]string)
}

func main() {
	http.HandleFunc("/", Shortener)
	http.ListenAndServe(":8080", nil)
}

package main

import (
	"io"
	"net/http"
	"strconv"
)

var baseUrl map[string]string
var lastId int

func Shortener(w http.ResponseWriter, r *http.Request) {
	// Обработка метода Post
	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Не удалось прочитать тело POST запроса.", http.StatusInternalServerError)
			return
		}
		lastId++
		shortKey := strconv.Itoa(lastId)
		baseUrl[shortKey] = string(body)
		// создание короткой ссылки из хоста и shortKey
		shortUrl := r.Host + "/" + shortKey
		w.WriteHeader(201)
		w.Write([]byte(shortUrl))
	}

	// Обработка метода Get
	if r.Method == http.MethodGet {
		shortKey := r.URL.Path[1:]
		fullUrl, ok := baseUrl[shortKey]
		if ok {
			w.Header().Set("Location", fullUrl)
			w.WriteHeader(307)
		} else {
			w.WriteHeader(400)
		}
	}

}

func init() {
	baseUrl = make(map[string]string)
}

func main() {
	http.HandleFunc("/", Shortener)
	http.ListenAndServe(":8080", nil)
}

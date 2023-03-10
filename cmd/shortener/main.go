package main

import (
	"log"
	"net/http"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/handlers"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/bubu256/go-url-shortener-server/pkg/storage"
)

func main() {
	cfg := config.New()
	cfg.LoadFromFlag() // загрузка параметров из флагов запуска или значения по умолчанию
	cfg.LoadFromEnv()  // загрузка параметров из переменных окружения
	dataStorage := storage.New(cfg.DB, nil)
	service := shortener.New(dataStorage, cfg.Service)
	handler := handlers.New(service, cfg.Server)
	log.Println("Сервер:", cfg.Server.ServerAddress)
	log.Fatal(http.ListenAndServe(cfg.Server.ServerAddress, handler.Router))
}

package main

import (
	"log"
	"net/http"

	"github.com/bubu256/go-url-shortener-server/internal/app/handlers"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/bubu256/go-url-shortener-server/pkg/config"
	"github.com/bubu256/go-url-shortener-server/pkg/storage"
)

/*
Новая структура, но неуверен что распределил пакеты в нужные папки

cmd\
└── main
internal\app
└── shortener - бизнес логика (создание ключа)
└── handlers - роутинг и обработчики запросов
pkg
└── storage - реализация БД (сейчас на sync.Map)
└── config - структура для хранение параметров программы (пока почти пустая)
*/

func main() {
	cfg := config.New()
	dataStorage := storage.New(cfg.DB)
	service := shortener.New(dataStorage)
	handler := handlers.New(service)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, handler.Router))
}

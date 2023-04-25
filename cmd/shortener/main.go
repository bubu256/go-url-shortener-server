package main

import (
	// "crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/handlers"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"github.com/bubu256/go-url-shortener-server/pkg/storage"

	// "golang.org/x/crypto/acme/autocert"

	// "github.com/bubu256/go-url-shortener-server/profiles"
	_ "net/http/pprof"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

// greeting - функция выводит версию, дату, и коммит сборки. Если данные отсутствуют то выводится "N/A"
func greeting() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func main() {
	greeting()
	cfg := config.New()
	cfg.LoadFromFlag() // загрузка параметров из флагов запуска или значения по умолчанию
	cfg.LoadFromEnv()  // загрузка параметров из переменных окружения
	dataStorage := storage.New(cfg.DB, nil)
	service := shortener.New(dataStorage, cfg.Service)
	handler := handlers.New(service, cfg.Server)
	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	if !cfg.Server.EnableHTTPS {
		log.Println("Сервер:", cfg.Server.ServerAddress)
		log.Fatal(http.ListenAndServe(cfg.Server.ServerAddress, handler.Router))
	} else {
		http.ListenAndServeTLS(":443", "server.crt", "server.key", handler.Router)
	}

}

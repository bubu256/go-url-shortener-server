package main

import (
	// "crypto/tls"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

// handleSignals - обрабатывает сигналы прерывания программы. Останавливает сервер. и освобождает ресурсы
func handleSignals(server *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	sig := <-sigCh
	log.Printf("Received signal: %s\n", sig)

	// Остановка сервера
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Failed to gracefully shutdown server: %v\n", err)
	}

	// тут можно добавить закрытие открытых ресурсов
	// на данный момент все ресурсы открываются и закрываются по запросам - освобождать нечего.
}

func main() {
	greeting()
	cfg := config.New()
	cfg.LoadConfiguration() // загружаем конфигурацию
	dataStorage := storage.New(cfg.DB, nil)
	service := shortener.New(dataStorage, cfg.Service)
	handler := handlers.New(service, cfg.Server)
	go func() {
		http.ListenAndServe(":6060", nil) // сервер для профилирования
	}()

	log.Println("Сервер:", cfg.Server.ServerAddress)
	server := &http.Server{}
	if !cfg.Server.EnableHTTPS {
		server = &http.Server{
			Addr:    cfg.Server.ServerAddress,
			Handler: handler.Router,
		}
		go func() {
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Failed to start HTTP server: %v\n", err)
			}
		}()
	} else {
		server = &http.Server{
			Addr:    ":443",
			Handler: handler.Router,
		}
		go func() {
			log.Println("TLS mode is on")
			if err := server.ListenAndServeTLS("server.crt", "server.key"); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("Failed to start HTTPS server: %v\n", err)
			}
		}()
	}
	handleSignals(server)
	log.Println("Сервер остановлен.")
}

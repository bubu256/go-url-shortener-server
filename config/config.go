// Package config defines application configuration.
package config

import (
	"flag"
	"log"
	"net/url"

	"github.com/caarlos0/env"
)

// Возвращает экземпляр конфигурации приложения.
func New() Configuration {
	cfg := Configuration{
		Server: CfgServer{ServerAddress: "localhost:8080", Scheme: "http"},
	}
	return cfg
}

// Configuration - конфигурация приложения.
type Configuration struct {
	DB      CfgDataBase
	Server  CfgServer
	Service CfgService
}

// CfgService - конфигурация сервиса.
type CfgService struct {
	// Переменная для хранения секретного ключа сервиса.
	SecretKey string `env:"KEY"`
}

// CfgDataBase - конфигурация базы данных.
type CfgDataBase struct {
	// Путь к файлу для хранилища.
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// Строка подключения к базе данных.
	DataBaseDSN string `env:"DATABASE_DSN"`
}

// CfgServer - конфигурация сервера.
type CfgServer struct {
	// Адрес сервера.
	ServerAddress string `env:"SERVER_ADDRESS"`
	// Используемая схема (http/https).
	Scheme string
	// Базовый URL для формирования короткой ссылки
	BaseURL string `env:"BASE_URL"`
}

// Заполняет конфиг из переменных окружения.
// Используемые переменные окружения
// FILE_STORAGE_PATH - путь к файлу с хранилищем
// SERVER_ADDRESS - адрес поднимаемого сервера, например "localhost:8080"
// BASE_URL - базовый адрес для коротких ссылок "http://localhost:8080"
// KEY - секретный ключ для генерации токенов
// DATABASE_DSN - строка подключения к базе данных
func (c *Configuration) LoadFromEnv() {
	err := env.Parse(&(c.Server))
	if err != nil {
		log.Println("не удалось загрузить конфигурацию сервера из переменных окружения;", err)
	}

	err = env.Parse(&(c.DB))
	if err != nil {
		log.Println("не удалось загрузить конфигурацию хранилища из переменных окружения;", err)
	}
}

// LoadFromFlag - считывает флаги запуска приложения.
func (c *Configuration) LoadFromFlag() {
	flag.StringVar(&(c.Server.ServerAddress), "a", "localhost:8080", "Address to start the server (SERVER_ADDRESS environment)")
	flag.StringVar(&(c.Server.BaseURL), "b", "", "Shortlink base address (BASE_URL environment)")
	flag.StringVar(&(c.DB.FileStoragePath), "f", "", "path to storage files (FILE_STORAGE_PATH environment)")
	flag.StringVar(&(c.DB.DataBaseDSN), "d", "", "connecting string to DB (DATABASE_DSN environment)")
	flag.StringVar(&(c.Service.SecretKey), "k", "", "Secret key for token generating")
	flag.Parse()

	// Проверка базового url. Устанавливаем если url не указан или он не валидный
	baseURL, err := url.Parse(c.Server.BaseURL)
	if err != nil || baseURL.Host == "" {
		// если не вышло создаем базовый url на основе адреса сервера и схемы из конфига
		baseURL.Scheme = c.Server.Scheme
		baseURL.Host = c.Server.ServerAddress
		c.Server.BaseURL = baseURL.String()
		log.Printf("Конфигурация: baseURL автоматически установлен %q", c.Server.BaseURL)
	}
}

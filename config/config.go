package config

import (
	"flag"
	"log"
	"net/url"

	"github.com/caarlos0/env"
)

func New() Configuration {
	cfg := Configuration{
		Server: CfgServer{ServerAddress: "localhost:8080", Scheme: "http"},
	}
	return cfg
}

type Configuration struct {
	DB      CfgDataBase
	Server  CfgServer
	Service CfgService
}

type CfgService struct {
	SecretKey string `env:"KEY"`
}

type CfgDataBase struct {
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DataBaseDSN     string `env:"DATABASE_DSN"`
}

type CfgServer struct {
	ServerAddress string `env:"SERVER_ADDRESS"`
	Scheme        string
	BaseURL       string `env:"BASE_URL"`
}

// Заполняет конфиг из переменных окружения
// используемые переменные окружения:
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

// функция парсит флаги запуска
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

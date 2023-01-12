package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env"
)

func New() Configuration {
	cfg := Configuration{
		Server: CfgServer{ServerAddress: "localhost:8080", Scheme: "http"},
	}
	return cfg
}

type Configuration struct {
	DB     CfgDataBase
	Server CfgServer
	// ... тут будут конфиги и для других модулей наверное
}

type CfgDataBase struct {
	InitialData     map[string]string
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// ... тут будут настройки для Базы данных
}

type CfgServer struct {
	ServerAddress string `env:"SERVER_ADDRESS"`
	Scheme        string
	BaseURL       string `env:"BASE_URL"`
	// ... тут будут остальные настройки для Сервера
}

// Заполняет конфиг из переменных окружения
// используемые переменные окружения:
// FILE_STORAGE_PATH - путь к файлу с хранилищем
// SERVER_ADDRESS - адрес поднимаемого сервера, например "localhost:8080"
// BASE_URL - базовый адрес для коротких ссылок "http://localhost:8080"
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
	flag.StringVar(&(c.Server.BaseURL), "b", "http://localhost:8080", "Shortlink base address (BASE_URL environment)")
	flag.StringVar(&(c.DB.FileStoragePath), "f", "", "path to storage files (FILE_STORAGE_PATH environment)")
	flag.Parse()
}

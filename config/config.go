package config

import (
	"log"

	"github.com/caarlos0/env"
)

func New() Configuration {
	return Configuration{
		Server: CfgServer{ServerAddress: "localhost:8080", Scheme: "http"},
	}
}

type Configuration struct {
	DB     CfgDataBase
	Server CfgServer
	// ... тут будут конфиги и для других модулей наверное
}

type CfgDataBase struct {
	InitialData     map[string]string
	FileStoragePath string `env:"FILE_STORAGE_PATH,required"`
	// ... тут будут настройки для Базы данных
}

type CfgServer struct {
	ServerAddress string `env:"SERVER_ADDRESS,required"`
	Scheme        string
	BaseURL       string `env:"BASE_URL"`
	// ... тут будут остальные настройки для Сервера
}

// используемые переменные окружения:
// FILE_STORAGE_PATH - путь к файлу с хранилищем
// SERVER_ADDRESS - адрес поднимаемого сервера, например "localhost:8080", обязательное заполнение
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

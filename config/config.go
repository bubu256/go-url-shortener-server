package config

import (
	"log"

	"github.com/caarlos0/env"
)

// здесь пока очень пусто, но наверное в будущем эта структура пригодится

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
	InitialData map[string]string
	// ... тут будут настройки для Базы данных
}

type CfgServer struct {
	ServerAddress string `env:"SERVER_ADDRESS,required"`
	Scheme        string
	BaseURL       string `env:"BASE_URL"`
	// ... тут будут остальные настройки для Сервера
}

func (c *Configuration) LoadFromEnv() {
	err := env.Parse(&(c.Server))
	if err != nil {
		log.Println("не удалось загрузить конфигурацию сервера из переменных окружения;", err)
	}
}

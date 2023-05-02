// Package config defines application configuration.
package config

import (
	"encoding/json"
	"flag"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/caarlos0/env"
)

// Возвращает экземпляр конфигурации приложения.
func New() Configuration {
	cfg := Configuration{
		Server: CfgServer{ServerAddress: "localhost", Scheme: "http"},
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
	BaseURL     string `env:"BASE_URL"`
	EnableHTTPS bool   `env:"ENABLE_HTTPS"`
}

// LoadConfiguration - заполняет структуру Configuration согласно приоритету (от меньшего к большему).
// - ReadConfigFile - загрузка из файла конфигурации указанного через флаг -c или env CONFIG
// - LoadFromFlag - загрузка из флагов запуска
// - LoadFromEnv - загрузка из переменных окружения
func (c *Configuration) LoadConfiguration() {
	var cfgFilePath string
	for i, arg := range os.Args {
		if (arg == "-c" || arg == "-config") && i+1 < len(os.Args) {
			cfgFilePath = os.Args[i+1]
			log.Println("Указан файл конфигурации через флаг -c")
			break
		}
	}
	if v, ok := os.LookupEnv("CONFIG"); ok {
		log.Println("Указан файл конфигурации через переменную окружения CONFIG")
		cfgFilePath = v
	}

	if cfgFilePath != "" {
		err := c.ReadConfigFile(cfgFilePath)
		if err != nil {
			log.Printf("ошибка при чтении файла конфигурации %v", err)
		}
	}
	c.LoadFromFlag()
	c.LoadFromEnv()

	if c.Server.ServerAddress == "" {
		c.Server.ServerAddress = "localhost:8080"
	}
	if !strings.Contains(c.Server.ServerAddress, ":") {
		c.Server.ServerAddress = c.Server.ServerAddress + ":8080"
	}
}

// LoadFromEnv - заполняет конфиг из переменных окружения.
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

// ReadConfigFile - читает файл конфигурации и заполняет поля структуры.
func (c *Configuration) ReadConfigFile(filePath string) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	type cfgJSON struct {
		ServerAddress   string `json:"server_address"`
		BaseURL         string `json:"base_url"`
		FileStoragePath string `json:"file_storage_path"`
		DataBaseDSN     string `json:"database_dsn"`
		EnableHTTPS     bool   `json:"enable_https"`
		SecretKey       string `json:"key"`
	}
	cfgFromFile := cfgJSON{}

	err = json.Unmarshal(file, &cfgFromFile)
	if err != nil {
		return err
	}
	c.Server.BaseURL = cfgFromFile.BaseURL
	c.Server.EnableHTTPS = cfgFromFile.EnableHTTPS
	c.Server.ServerAddress = cfgFromFile.ServerAddress
	c.Service.SecretKey = cfgFromFile.SecretKey
	c.DB.DataBaseDSN = cfgFromFile.DataBaseDSN
	c.DB.FileStoragePath = cfgFromFile.FileStoragePath

	if c.Server.EnableHTTPS {
		c.Server.Scheme = "https"
	} else {
		c.Server.Scheme = "http"
	}

	return nil
}

// LoadFromFlag - считывает флаги запуска приложения.
func (c *Configuration) LoadFromFlag() {
	flag.StringVar(&(c.Server.ServerAddress), "a", c.Server.ServerAddress, "Address to start the server (SERVER_ADDRESS environment)")
	flag.StringVar(&(c.Server.BaseURL), "b", c.Server.BaseURL, "Shortlink base address (BASE_URL environment)")
	flag.StringVar(&(c.DB.FileStoragePath), "f", c.DB.FileStoragePath, "path to storage files (FILE_STORAGE_PATH environment)")
	flag.StringVar(&(c.DB.DataBaseDSN), "d", c.DB.DataBaseDSN, "connecting string to DB (DATABASE_DSN environment)")
	flag.StringVar(&(c.Service.SecretKey), "k", c.Service.SecretKey, "Secret key for token generating")
	flag.BoolVar(&(c.Server.EnableHTTPS), "s", c.Server.EnableHTTPS, "")
	flag.String("c", "", "path to the configuration file")
	flag.String("config", "", "path to the configuration file")
	flag.Parse()

	if c.Server.EnableHTTPS {
		c.Server.Scheme = "https"
		c.Server.ServerAddress = "localhost"
	}
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

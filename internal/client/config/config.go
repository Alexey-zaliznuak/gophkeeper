// Package config содержит конфигурацию клиентского приложения.
package config

import (
	"os"
	"path/filepath"
)

// Config содержит все настройки клиента.
type Config struct {
	// ServerAddress - адрес gRPC сервера.
	ServerAddress string

	// DataDir - директория для хранения локальных данных.
	DataDir string

	// TLSCertFile - путь к файлу CA сертификата для проверки сервера.
	TLSCertFile string

	// LogLevel - уровень логирования (debug, info, warn, error).
	LogLevel string
}

// Load загружает конфигурацию клиента.
// Конфигурация берётся из переменных окружения или используются значения по умолчанию.
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	cfg := &Config{
		ServerAddress: "localhost:3200",
		DataDir:       filepath.Join(homeDir, ".gophkeeper"),
		LogLevel:      "info",
	}

	// Переопределение из переменных окружения
	if addr := os.Getenv("GOPHKEEPER_SERVER_ADDRESS"); addr != "" {
		cfg.ServerAddress = addr
	}
	if dataDir := os.Getenv("GOPHKEEPER_DATA_DIR"); dataDir != "" {
		cfg.DataDir = dataDir
	}
	if cert := os.Getenv("GOPHKEEPER_TLS_CERT"); cert != "" {
		cfg.TLSCertFile = cert
	}
	if level := os.Getenv("GOPHKEEPER_LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	// Создание директории для данных, если не существует
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, err
	}

	return cfg, nil
}

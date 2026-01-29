// Package config содержит конфигурацию серверного приложения.
package config

import (
	"flag"
	"os"
	"time"
)

// Config содержит все настройки сервера.
type Config struct {
	// Address - адрес для прослушивания gRPC сервером.
	Address string

	// DatabaseDSN - строка подключения к PostgreSQL.
	DatabaseDSN string

	// JWTSecret - секретный ключ для подписи JWT токенов.
	JWTSecret string

	// AccessTokenTTL - время жизни access токена.
	AccessTokenTTL time.Duration

	// RefreshTokenTTL - время жизни refresh токена.
	RefreshTokenTTL time.Duration

	// TLSCertFile - путь к файлу TLS сертификата.
	TLSCertFile string

	// TLSKeyFile - путь к файлу приватного ключа TLS.
	TLSKeyFile string

	// LogLevel - уровень логирования (debug, info, warn, error).
	LogLevel string
}

// Load загружает конфигурацию из флагов командной строки и переменных окружения.
// Приоритет: флаги > переменные окружения > значения по умолчанию.
func Load() (*Config, error) {
	cfg := &Config{}

	// Определение флагов
	flag.StringVar(&cfg.Address, "address", ":3200", "gRPC server address")
	flag.StringVar(&cfg.DatabaseDSN, "database-dsn", "", "PostgreSQL connection string")
	flag.StringVar(&cfg.JWTSecret, "jwt-secret", "", "JWT signing secret")
	flag.DurationVar(&cfg.AccessTokenTTL, "access-token-ttl", 15*time.Minute, "Access token TTL")
	flag.DurationVar(&cfg.RefreshTokenTTL, "refresh-token-ttl", 7*24*time.Hour, "Refresh token TTL")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", "", "TLS certificate file path")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", "", "TLS private key file path")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	flag.Parse()

	// Переопределение из переменных окружения
	if addr := os.Getenv("GOPHKEEPER_ADDRESS"); addr != "" {
		cfg.Address = addr
	}
	if dsn := os.Getenv("GOPHKEEPER_DATABASE_DSN"); dsn != "" {
		cfg.DatabaseDSN = dsn
	}
	if secret := os.Getenv("GOPHKEEPER_JWT_SECRET"); secret != "" {
		cfg.JWTSecret = secret
	}
	if cert := os.Getenv("GOPHKEEPER_TLS_CERT"); cert != "" {
		cfg.TLSCertFile = cert
	}
	if key := os.Getenv("GOPHKEEPER_TLS_KEY"); key != "" {
		cfg.TLSKeyFile = key
	}
	if level := os.Getenv("GOPHKEEPER_LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	return cfg, nil
}

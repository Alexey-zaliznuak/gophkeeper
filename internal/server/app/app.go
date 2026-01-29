// Package app содержит логику инициализации и запуска серверного приложения.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"gophkeeper/internal/server/config"
	"gophkeeper/internal/server/grpc"
	"gophkeeper/internal/server/repository/postgres"
	"gophkeeper/internal/server/service/auth"
	"gophkeeper/internal/server/service/secret"
	syncService "gophkeeper/internal/server/service/sync"
	"gophkeeper/pkg/jwt"
	"gophkeeper/pkg/logger"
)

// Run запускает серверное приложение GophKeeper.
// Инициализирует конфигурацию, логгер, подключения к БД и запускает gRPC сервер.
func Run(version, buildDate string) error {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Инициализация логгера
	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	defer log.Sync()

	log.Info("starting GophKeeper server",
		zap.String("version", version),
		zap.String("build_date", buildDate),
	)

	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		log.Warn("WARNING: TLS is not configured! Server is running WITHOUT encryption. " +
			"This is insecure and should only be used for development. " +
			"Set GOPHKEEPER_TLS_CERT and GOPHKEEPER_TLS_KEY for production.")
	}

	// Контекст с отменой для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Подключение к базе данных
	if cfg.DatabaseDSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	db, err := postgres.New(ctx, cfg.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	log.Info("connected to database")

	// Инициализация репозиториев
	userRepo := postgres.NewUserRepository(db)
	secretRepo := postgres.NewSecretRepository(db)
	tokenRepo := postgres.NewRefreshTokenRepository(db)

	// Инициализация JWT менеджера
	if cfg.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL)

	// Инициализация сервисов
	authService := auth.NewService(userRepo, tokenRepo, jwtManager, cfg.RefreshTokenTTL, log)
	secretService := secret.NewService(secretRepo, log)
	syncSvc := syncService.NewService(secretRepo, log)

	// Создание gRPC сервера
	grpcServer, err := grpc.NewServer(
		grpc.Config{
			Address:     cfg.Address,
			TLSCertFile: cfg.TLSCertFile,
			TLSKeyFile:  cfg.TLSKeyFile,
		},
		grpc.Services{
			Auth:   authService,
			Secret: secretService,
			Sync:   syncSvc,
		},
		log,
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC server: %w", err)
	}

	// Обработка сигналов завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в горутине
	errCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Start(); err != nil {
			errCh <- err
		}
	}()

	// Ожидание сигнала или ошибки
	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		log.Info("received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	grpcServer.Stop()
	log.Info("server stopped")

	return nil
}

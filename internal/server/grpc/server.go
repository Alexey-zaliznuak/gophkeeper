// Package grpc содержит gRPC сервер и обработчики.
package grpc

import (
	"crypto/tls"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"gophkeeper/internal/server/grpc/handler"
	"gophkeeper/internal/server/grpc/interceptor"
	"gophkeeper/internal/server/grpc/pb"
	"gophkeeper/internal/server/service"
)

// Server представляет gRPC сервер.
type Server struct {
	server   *grpc.Server
	listener net.Listener
	log      *zap.Logger
}

// Config конфигурация gRPC сервера.
type Config struct {
	Address     string
	TLSCertFile string
	TLSKeyFile  string
}

// Services содержит сервисы для обработчиков.
type Services struct {
	Auth   service.AuthService
	Secret service.SecretService
	Sync   service.SyncService
}

// NewServer создаёт новый gRPC сервер.
func NewServer(cfg Config, services Services, log *zap.Logger) (*Server, error) {
	// Создаём listener
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	// Настраиваем interceptors
	loggingInterceptor := interceptor.NewLoggingInterceptor(log)
	recoveryInterceptor := interceptor.NewRecoveryInterceptor(log)
	authInterceptor := interceptor.NewAuthInterceptor(services.Auth, log)

	// Опции сервера
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor.Unary(),
			loggingInterceptor.Unary(),
			authInterceptor.Unary(),
		),
		grpc.ChainStreamInterceptor(
			recoveryInterceptor.Stream(),
			loggingInterceptor.Stream(),
			authInterceptor.Stream(),
		),
	}

	// Настраиваем TLS если указаны сертификаты
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		log.Info("TLS enabled")
	} else {
		log.Warn("TLS disabled - running in insecure mode")
	}

	// Создаём gRPC сервер
	server := grpc.NewServer(opts...)

	// Регистрируем обработчики
	pb.RegisterAuthServiceServer(server, handler.NewAuthHandler(services.Auth))
	pb.RegisterSecretServiceServer(server, handler.NewSecretHandler(services.Secret))
	pb.RegisterSyncServiceServer(server, handler.NewSyncHandler(services.Sync))

	return &Server{
		server:   server,
		listener: listener,
		log:      log,
	}, nil
}

// Start запускает gRPC сервер.
func (s *Server) Start() error {
	s.log.Info("starting gRPC server", zap.String("address", s.listener.Addr().String()))
	return s.server.Serve(s.listener)
}

// Stop останавливает gRPC сервер.
func (s *Server) Stop() {
	s.log.Info("stopping gRPC server")
	s.server.GracefulStop()
}

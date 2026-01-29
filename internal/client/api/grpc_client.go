// Package api содержит gRPC клиент для связи с сервером.
package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"gophkeeper/internal/server/grpc/pb"
)

// Client gRPC клиент для взаимодействия с сервером.
type Client struct {
	conn         *grpc.ClientConn
	authClient   pb.AuthServiceClient
	secretClient pb.SecretServiceClient
	syncClient   pb.SyncServiceClient
	accessToken  string
}

// NewClient создаёт новый gRPC клиент.
func NewClient(serverAddr string, tlsCertFile string) (*Client, error) {
	var opts []grpc.DialOption

	if tlsCertFile != "" {
		// Загружаем CA сертификат
		caCert, err := os.ReadFile(tlsCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to add CA cert to pool")
		}

		tlsConfig := &tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS12,
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(serverAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Client{
		conn:         conn,
		authClient:   pb.NewAuthServiceClient(conn),
		secretClient: pb.NewSecretServiceClient(conn),
		syncClient:   pb.NewSyncServiceClient(conn),
	}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error {
	return c.conn.Close()
}

// SetAccessToken устанавливает токен для авторизации запросов.
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// GetAccessToken возвращает текущий access токен.
func (c *Client) GetAccessToken() string {
	return c.accessToken
}

// authContext добавляет токен авторизации в контекст.
func (c *Client) authContext(ctx context.Context) context.Context {
	if c.accessToken == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.accessToken)
}

// ==================== Auth ====================

// Register регистрирует нового пользователя.
func (c *Client) Register(ctx context.Context, login, password string) (*pb.RegisterResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return c.authClient.Register(ctx, &pb.RegisterRequest{
		Login:    login,
		Password: password,
	})
}

// Login выполняет вход пользователя.
func (c *Client) Login(ctx context.Context, login, password string) (*pb.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return c.authClient.Login(ctx, &pb.LoginRequest{
		Login:    login,
		Password: password,
	})
}

// RefreshToken обновляет пару токенов.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return c.authClient.RefreshToken(ctx, &pb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
}

// Logout выполняет выход пользователя.
func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.authClient.Logout(ctx, &pb.LogoutRequest{
		RefreshToken: refreshToken,
	})
	return err
}

// ==================== Secrets ====================

// CreateSecret создаёт новый секрет.
func (c *Client) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	ctx, cancel := context.WithTimeout(c.authContext(ctx), 30*time.Second)
	defer cancel()

	return c.secretClient.Create(ctx, req)
}

// GetSecret получает секрет по ID.
func (c *Client) GetSecret(ctx context.Context, id string) (*pb.GetSecretResponse, error) {
	ctx, cancel := context.WithTimeout(c.authContext(ctx), 10*time.Second)
	defer cancel()

	return c.secretClient.Get(ctx, &pb.GetSecretRequest{Id: id})
}

// UpdateSecret обновляет секрет.
func (c *Client) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	ctx, cancel := context.WithTimeout(c.authContext(ctx), 30*time.Second)
	defer cancel()

	return c.secretClient.Update(ctx, req)
}

// DeleteSecret удаляет секрет.
func (c *Client) DeleteSecret(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c.authContext(ctx), 10*time.Second)
	defer cancel()

	_, err := c.secretClient.Delete(ctx, &pb.DeleteSecretRequest{Id: id})
	return err
}

// ListSecrets возвращает список всех секретов.
func (c *Client) ListSecrets(ctx context.Context) (*pb.ListSecretsResponse, error) {
	ctx, cancel := context.WithTimeout(c.authContext(ctx), 30*time.Second)
	defer cancel()

	return c.secretClient.List(ctx, &pb.ListSecretsRequest{})
}

// ==================== Sync ====================

// GetChanges получает изменения с сервера.
func (c *Client) GetChanges(ctx context.Context, req *pb.GetChangesRequest) (*pb.GetChangesResponse, error) {
	ctx, cancel := context.WithTimeout(c.authContext(ctx), 60*time.Second)
	defer cancel()

	return c.syncClient.GetChanges(ctx, req)
}

// PushChanges отправляет изменения на сервер.
func (c *Client) PushChanges(ctx context.Context, req *pb.PushChangesRequest) (*pb.PushChangesResponse, error) {
	ctx, cancel := context.WithTimeout(c.authContext(ctx), 60*time.Second)
	defer cancel()

	return c.syncClient.PushChanges(ctx, req)
}

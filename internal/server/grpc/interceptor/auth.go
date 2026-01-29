// Package interceptor содержит gRPC interceptors.
package interceptor

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"gophkeeper/internal/server/service"
)

// ContextKey тип для ключей контекста.
type ContextKey string

const (
	// UserIDKey ключ для ID пользователя в контексте.
	UserIDKey ContextKey = "user_id"

	// authorizationHeader заголовок для токена.
	authorizationHeader = "authorization"

	// bearerPrefix префикс Bearer токена.
	bearerPrefix = "Bearer "
)

// AuthInterceptor interceptor для проверки аутентификации.
type AuthInterceptor struct {
	authService service.AuthService
	log         *zap.Logger
	// Методы, не требующие аутентификации
	publicMethods map[string]bool
}

// NewAuthInterceptor создаёт новый interceptor аутентификации.
func NewAuthInterceptor(authService service.AuthService, log *zap.Logger) *AuthInterceptor {
	return &AuthInterceptor{
		authService: authService,
		log:         log,
		publicMethods: map[string]bool{
			"/gophkeeper.api.v1.AuthService/Register":     true,
			"/gophkeeper.api.v1.AuthService/Login":        true,
			"/gophkeeper.api.v1.AuthService/RefreshToken": true,
			"/gophkeeper.api.v1.AuthService/Logout":       true,
		},
	}
}

// Unary возвращает unary interceptor.
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Проверяем, требуется ли аутентификация
		if i.publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Извлекаем и проверяем токен
		userID, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}

		// Добавляем userID в контекст
		ctx = context.WithValue(ctx, UserIDKey, userID)

		return handler(ctx, req)
	}
}

// Stream возвращает stream interceptor.
func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Проверяем, требуется ли аутентификация
		if i.publicMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		// Извлекаем и проверяем токен
		userID, err := i.authenticate(ss.Context())
		if err != nil {
			return err
		}

		// Создаём обёртку стрима с новым контекстом
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), UserIDKey, userID),
		}

		return handler(srv, wrappedStream)
	}
}

// authenticate извлекает и проверяет токен из метаданных.
func (i *AuthInterceptor) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get(authorizationHeader)
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := values[0]
	if !strings.HasPrefix(token, bearerPrefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header format")
	}

	accessToken := strings.TrimPrefix(token, bearerPrefix)

	userID, err := i.authService.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		i.log.Debug("authentication failed", zap.Error(err))
		return "", status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	return userID, nil
}

// wrappedServerStream обёртка для ServerStream с кастомным контекстом.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context возвращает контекст с userID.
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// GetUserIDFromContext извлекает ID пользователя из контекста.
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", status.Error(codes.Internal, "user id not found in context")
	}
	return userID, nil
}

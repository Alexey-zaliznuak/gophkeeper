// Package handler содержит gRPC обработчики.
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gophkeeper/internal/model"
	"gophkeeper/internal/server/grpc/pb"
	"gophkeeper/internal/server/service"
)

// AuthHandler обработчик AuthService.
type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	authService service.AuthService
}

// NewAuthHandler создаёт новый обработчик аутентификации.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register регистрирует нового пользователя.
func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password are required")
	}

	user, tokens, err := h.authService.Register(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		return nil, mapAuthError(err)
	}

	return pb.RegisterResponse_builder{
		UserId:       user.ID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}.Build(), nil
}

// Login выполняет аутентификацию пользователя.
func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.GetLogin() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password are required")
	}

	user, tokens, err := h.authService.Login(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		return nil, mapAuthError(err)
	}

	return pb.LoginResponse_builder{
		UserId:       user.ID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}.Build(), nil
}

// RefreshToken обновляет пару токенов.
func (h *AuthHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	tokens, err := h.authService.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, mapAuthError(err)
	}

	return pb.RefreshTokenResponse_builder{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}.Build(), nil
}

// Logout завершает сессию пользователя.
func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	if err := h.authService.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, mapAuthError(err)
	}

	return pb.LogoutResponse_builder{}.Build(), nil
}

// mapAuthError преобразует ошибки аутентификации в gRPC статусы.
func mapAuthError(err error) error {
	switch err {
	case model.ErrUserAlreadyExists:
		return status.Error(codes.AlreadyExists, "user already exists")
	case model.ErrUserNotFound, model.ErrInvalidCredentials:
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case model.ErrInvalidToken:
		return status.Error(codes.Unauthenticated, "invalid token")
	case model.ErrTokenExpired:
		return status.Error(codes.Unauthenticated, "token expired")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

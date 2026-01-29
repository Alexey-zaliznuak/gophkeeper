// Package auth реализует сервис аутентификации.
package auth

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"gophkeeper/internal/model"
	"gophkeeper/internal/server/repository"
	"gophkeeper/pkg/crypto"
	"gophkeeper/pkg/jwt"
)

// Service реализует сервис аутентификации.
type Service struct {
	userRepo   repository.UserRepository
	tokenRepo  repository.RefreshTokenRepository
	jwtManager *jwt.Manager
	refreshTTL time.Duration
	log        *zap.Logger
}

// NewService создаёт новый сервис аутентификации.
func NewService(
	userRepo repository.UserRepository,
	tokenRepo repository.RefreshTokenRepository,
	jwtManager *jwt.Manager,
	refreshTTL time.Duration,
	log *zap.Logger,
) *Service {
	return &Service{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtManager: jwtManager,
		refreshTTL: refreshTTL,
		log:        log,
	}
}

// Register регистрирует нового пользователя и возвращает пару токенов.
func (s *Service) Register(ctx context.Context, login, password string) (*model.User, *model.TokenPair, error) {
	// Хешируем пароль
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		s.log.Error("failed to hash password", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаём пользователя
	user := &model.User{
		Login:        login,
		PasswordHash: passwordHash,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error("failed to create user", zap.String("login", login), zap.Error(err))
		return nil, nil, err
	}

	s.log.Info("user registered", zap.String("user_id", user.ID), zap.String("login", login))

	// Генерируем токены
	tokens, err := s.generateTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// Login выполняет аутентификацию и возвращает пару токенов.
func (s *Service) Login(ctx context.Context, login, password string) (*model.User, *model.TokenPair, error) {
	// Ищем пользователя
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		if err == model.ErrUserNotFound {
			return nil, nil, model.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	// Проверяем пароль
	if !crypto.CheckPassword(password, user.PasswordHash) {
		return nil, nil, model.ErrInvalidCredentials
	}

	s.log.Info("user logged in", zap.String("user_id", user.ID), zap.String("login", login))

	// Генерируем токены
	tokens, err := s.generateTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshToken обновляет пару токенов по refresh токену.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	// Хешируем токен для поиска
	tokenHash := crypto.HashToken(refreshToken)

	// Ищем токен в БД
	storedToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, model.ErrInvalidToken
	}

	// Проверяем срок действия
	if time.Now().After(storedToken.ExpiresAt) {
		// Удаляем истёкший токен
		_ = s.tokenRepo.Delete(ctx, storedToken.ID)
		return nil, model.ErrTokenExpired
	}

	// Удаляем старый токен
	if err := s.tokenRepo.Delete(ctx, storedToken.ID); err != nil {
		s.log.Warn("failed to delete old refresh token", zap.Error(err))
	}

	// Генерируем новые токены
	return s.generateTokens(ctx, storedToken.UserID)
}

// Logout инвалидирует refresh токен.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := crypto.HashToken(refreshToken)

	storedToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil // Токен уже не существует - ничего делать не надо
	}

	return s.tokenRepo.Delete(ctx, storedToken.ID)
}

// ValidateAccessToken проверяет access токен и возвращает ID пользователя.
func (s *Service) ValidateAccessToken(ctx context.Context, accessToken string) (string, error) {
	userID, err := s.jwtManager.GetUserID(accessToken)
	if err != nil {
		return "", model.ErrInvalidToken
	}
	return userID, nil
}

// generateTokens создаёт пару access/refresh токенов.
func (s *Service) generateTokens(ctx context.Context, userID string) (*model.TokenPair, error) {
	// Генерируем access токен
	accessToken, err := s.jwtManager.Generate(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Генерируем refresh токен
	refreshTokenStr, err := crypto.GenerateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохраняем refresh токен в БД
	refreshToken := &model.RefreshToken{
		UserID:    userID,
		TokenHash: crypto.HashToken(refreshTokenStr),
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
	}, nil
}

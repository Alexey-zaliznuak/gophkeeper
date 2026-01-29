// Package jwt предоставляет функции для работы с JWT токенами.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims содержит данные, хранящиеся в JWT токене.
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
}

// Manager управляет созданием и валидацией JWT токенов.
type Manager struct {
	secret   []byte
	tokenTTL time.Duration
}

// NewManager создаёт новый менеджер JWT токенов.
func NewManager(secret string, tokenTTL time.Duration) *Manager {
	return &Manager{
		secret:   []byte(secret),
		tokenTTL: tokenTTL,
	}
}

// Generate создаёт новый JWT токен для пользователя.
func (m *Manager) Generate(userID string) (string, error) {
	now := time.Now()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gophkeeper",
		},
		UserID: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// Validate проверяет JWT токен и возвращает claims.
func (m *Manager) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GetUserID извлекает ID пользователя из токена.
func (m *Manager) GetUserID(tokenString string) (string, error) {
	claims, err := m.Validate(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

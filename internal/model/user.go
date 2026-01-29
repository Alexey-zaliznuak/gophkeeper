// Package model содержит доменные модели приложения.
package model

import (
	"time"
)

// User представляет пользователя системы.
type User struct {
	// ID - уникальный идентификатор пользователя (UUID).
	ID string

	// Login - уникальный логин пользователя.
	Login string

	// PasswordHash - хеш пароля (bcrypt).
	PasswordHash string

	// CreatedAt - дата создания учётной записи.
	CreatedAt time.Time

	// UpdatedAt - дата последнего обновления.
	UpdatedAt time.Time
}

// RefreshToken представляет refresh токен для обновления сессии.
type RefreshToken struct {
	// ID - уникальный идентификатор токена.
	ID string

	// UserID - ID пользователя-владельца токена.
	UserID string

	// TokenHash - хеш токена для безопасного хранения.
	TokenHash string

	// ExpiresAt - время истечения токена.
	ExpiresAt time.Time

	// CreatedAt - дата создания токена.
	CreatedAt time.Time
}

// TokenPair содержит пару access и refresh токенов.
type TokenPair struct {
	// AccessToken - токен для авторизации запросов.
	AccessToken string

	// RefreshToken - токен для обновления access токена.
	RefreshToken string
}

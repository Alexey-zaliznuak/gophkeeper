// Package repository содержит интерфейсы и реализации для работы с хранилищем данных.
package repository

import (
	"context"
	"time"

	"gophkeeper/internal/model"
)

// UserRepository определяет интерфейс для работы с пользователями.
type UserRepository interface {
	// Create создаёт нового пользователя.
	Create(ctx context.Context, user *model.User) error

	// GetByID возвращает пользователя по ID.
	GetByID(ctx context.Context, id string) (*model.User, error)

	// GetByLogin возвращает пользователя по логину.
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}

// SecretRepository определяет интерфейс для работы с секретами.
type SecretRepository interface {
	// Create создаёт новый секрет.
	Create(ctx context.Context, secret *model.Secret) error

	// Update обновляет существующий секрет.
	Update(ctx context.Context, secret *model.Secret) error

	// Delete выполняет soft delete секрета.
	Delete(ctx context.Context, userID, secretID string) error

	// GetByID возвращает секрет по ID.
	GetByID(ctx context.Context, userID, secretID string) (*model.Secret, error)

	// GetByName возвращает секрет по имени.
	GetByName(ctx context.Context, userID, name string) (*model.Secret, error)

	// List возвращает все секреты пользователя.
	List(ctx context.Context, userID string) ([]*model.Secret, error)

	// ListChangedSince возвращает секреты, изменённые после указанного времени.
	ListChangedSince(ctx context.Context, userID string, since time.Time) ([]*model.Secret, error)

	// ListDeletedSince возвращает ID удалённых секретов после указанного времени.
	ListDeletedSince(ctx context.Context, userID string, since time.Time) ([]string, error)
}

// RefreshTokenRepository определяет интерфейс для работы с refresh токенами.
type RefreshTokenRepository interface {
	// Create создаёт новый refresh токен.
	Create(ctx context.Context, token *model.RefreshToken) error

	// GetByTokenHash возвращает токен по хешу.
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	// Delete удаляет токен.
	Delete(ctx context.Context, id string) error

	// DeleteByUserID удаляет все токены пользователя.
	DeleteByUserID(ctx context.Context, userID string) error

	// DeleteExpired удаляет все истёкшие токены.
	DeleteExpired(ctx context.Context) error
}

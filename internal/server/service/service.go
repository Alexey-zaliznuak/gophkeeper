// Package service содержит интерфейсы и реализации бизнес-логики.
package service

import (
	"context"
	"time"

	"gophkeeper/internal/model"
)

// AuthService определяет интерфейс сервиса аутентификации.
type AuthService interface {
	// Register регистрирует нового пользователя и возвращает пару токенов.
	Register(ctx context.Context, login, password string) (*model.User, *model.TokenPair, error)

	// Login выполняет аутентификацию и возвращает пару токенов.
	Login(ctx context.Context, login, password string) (*model.User, *model.TokenPair, error)

	// RefreshToken обновляет пару токенов по refresh токену.
	RefreshToken(ctx context.Context, refreshToken string) (*model.TokenPair, error)

	// Logout инвалидирует refresh токен.
	Logout(ctx context.Context, refreshToken string) error

	// ValidateAccessToken проверяет access токен и возвращает ID пользователя.
	ValidateAccessToken(ctx context.Context, accessToken string) (string, error)
}

// SecretService определяет интерфейс сервиса работы с секретами.
type SecretService interface {
	// Create создаёт новый секрет.
	Create(ctx context.Context, userID string, secret *model.Secret) (*model.Secret, error)

	// Update обновляет существующий секрет.
	Update(ctx context.Context, userID string, secret *model.Secret, expectedVersion int64) (*model.Secret, error)

	// Delete удаляет секрет.
	Delete(ctx context.Context, userID, secretID string) error

	// Get возвращает секрет по ID.
	Get(ctx context.Context, userID, secretID string) (*model.Secret, error)

	// List возвращает все секреты пользователя.
	List(ctx context.Context, userID string) ([]*model.Secret, error)
}

// SyncService определяет интерфейс сервиса синхронизации.
type SyncService interface {
	// GetChanges возвращает изменения с указанного времени.
	GetChanges(ctx context.Context, userID string, since *time.Time) (*SyncChanges, error)

	// PushChanges применяет изменения от клиента.
	PushChanges(ctx context.Context, userID string, changes []*SecretChange) ([]*ChangeResult, error)
}

// SyncChanges содержит результат запроса изменений.
type SyncChanges struct {
	// Secrets - изменённые или созданные секреты.
	Secrets []*model.Secret

	// DeletedIDs - ID удалённых секретов.
	DeletedIDs []string

	// ServerTime - текущее время сервера.
	ServerTime time.Time
}

// ChangeOperation определяет тип операции изменения.
type ChangeOperation int

const (
	// ChangeOperationCreate - создание.
	ChangeOperationCreate ChangeOperation = iota + 1
	// ChangeOperationUpdate - обновление.
	ChangeOperationUpdate
	// ChangeOperationDelete - удаление.
	ChangeOperationDelete
)

// SecretChange представляет изменение от клиента.
type SecretChange struct {
	// Operation - тип операции.
	Operation ChangeOperation

	// Secret - данные секрета (для Create/Update).
	Secret *model.Secret

	// SecretID - ID секрета (для Delete).
	SecretID string

	// ClientID - локальный ID на клиенте.
	ClientID string
}

// ChangeResult представляет результат применения изменения.
type ChangeResult struct {
	// ClientID - локальный ID клиента.
	ClientID string

	// ServerID - ID на сервере.
	ServerID string

	// Success - успешно ли применено.
	Success bool

	// ErrorMessage - сообщение об ошибке.
	ErrorMessage string

	// ConflictSecret - конфликтующий секрет (при конфликте версий).
	ConflictSecret *model.Secret
}

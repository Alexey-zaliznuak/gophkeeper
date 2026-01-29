package model

import (
	"time"
)

// SecretType определяет тип секрета.
type SecretType int

const (
	// SecretTypeUnspecified - тип не указан.
	SecretTypeUnspecified SecretType = iota

	// SecretTypeCredentials - пара логин/пароль.
	SecretTypeCredentials

	// SecretTypeText - текстовые данные.
	SecretTypeText

	// SecretTypeBinary - бинарные данные.
	SecretTypeBinary

	// SecretTypeCard - данные банковской карты.
	SecretTypeCard
)

// String возвращает строковое представление типа секрета.
func (t SecretType) String() string {
	switch t {
	case SecretTypeCredentials:
		return "credentials"
	case SecretTypeText:
		return "text"
	case SecretTypeBinary:
		return "binary"
	case SecretTypeCard:
		return "card"
	default:
		return "unknown"
	}
}

// Secret представляет секрет пользователя.
type Secret struct {
	// ID - уникальный идентификатор секрета (UUID).
	ID string

	// UserID - ID пользователя-владельца.
	UserID string

	// Name - название секрета для отображения.
	Name string

	// Type - тип секрета.
	Type SecretType

	// EncryptedData - зашифрованные данные секрета.
	EncryptedData []byte

	// Metadata - дополнительные метаданные (сайт, заметки и т.д.).
	Metadata map[string]string

	// Version - версия для оптимистичной блокировки.
	Version int64

	// CreatedAt - дата создания.
	CreatedAt time.Time

	// UpdatedAt - дата последнего обновления.
	UpdatedAt time.Time

	// DeletedAt - дата удаления (soft delete), nil если не удалён.
	DeletedAt *time.Time
}

// IsDeleted возвращает true, если секрет помечен как удалённый.
func (s *Secret) IsDeleted() bool {
	return s.DeletedAt != nil
}

package model

import (
	"testing"
	"time"
)

func TestSecretType_String(t *testing.T) {
	tests := []struct {
		name       string
		secretType SecretType
		want       string
	}{
		{"credentials", SecretTypeCredentials, "credentials"},
		{"text", SecretTypeText, "text"},
		{"binary", SecretTypeBinary, "binary"},
		{"card", SecretTypeCard, "card"},
		{"unspecified", SecretTypeUnspecified, "unknown"},
		{"invalid", SecretType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.secretType.String()
			if got != tt.want {
				t.Errorf("SecretType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecret_IsDeleted(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		deletedAt *time.Time
		want      bool
	}{
		{"not deleted", nil, false},
		{"deleted", &now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Secret{DeletedAt: tt.deletedAt}
			got := s.IsDeleted()
			if got != tt.want {
				t.Errorf("Secret.IsDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecretTypeConstants(t *testing.T) {
	// Проверяем что константы имеют ожидаемые значения
	if SecretTypeUnspecified != 0 {
		t.Errorf("SecretTypeUnspecified = %d, want 0", SecretTypeUnspecified)
	}
	if SecretTypeCredentials != 1 {
		t.Errorf("SecretTypeCredentials = %d, want 1", SecretTypeCredentials)
	}
	if SecretTypeText != 2 {
		t.Errorf("SecretTypeText = %d, want 2", SecretTypeText)
	}
	if SecretTypeBinary != 3 {
		t.Errorf("SecretTypeBinary = %d, want 3", SecretTypeBinary)
	}
	if SecretTypeCard != 4 {
		t.Errorf("SecretTypeCard = %d, want 4", SecretTypeCard)
	}
}

func TestSecret_Fields(t *testing.T) {
	now := time.Now()
	metadata := map[string]string{"key": "value"}

	secret := &Secret{
		ID:            "test-id",
		UserID:        "user-id",
		Name:          "test secret",
		Type:          SecretTypeCredentials,
		EncryptedData: []byte("encrypted"),
		Metadata:      metadata,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
		DeletedAt:     nil,
	}

	if secret.ID != "test-id" {
		t.Errorf("Secret.ID = %v, want test-id", secret.ID)
	}
	if secret.UserID != "user-id" {
		t.Errorf("Secret.UserID = %v, want user-id", secret.UserID)
	}
	if secret.Name != "test secret" {
		t.Errorf("Secret.Name = %v, want test secret", secret.Name)
	}
	if secret.Type != SecretTypeCredentials {
		t.Errorf("Secret.Type = %v, want SecretTypeCredentials", secret.Type)
	}
	if string(secret.EncryptedData) != "encrypted" {
		t.Errorf("Secret.EncryptedData = %v, want encrypted", secret.EncryptedData)
	}
	if secret.Metadata["key"] != "value" {
		t.Errorf("Secret.Metadata[key] = %v, want value", secret.Metadata["key"])
	}
	if secret.Version != 1 {
		t.Errorf("Secret.Version = %v, want 1", secret.Version)
	}
}

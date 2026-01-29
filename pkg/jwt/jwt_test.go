package jwt

import (
	"testing"
	"time"
)

func TestManager_Generate(t *testing.T) {
	manager := NewManager("test-secret-key", time.Hour)

	tests := []struct {
		name   string
		userID string
	}{
		{"simple user id", "user123"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"empty user id", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.Generate(tt.userID)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if token == "" {
				t.Error("Generate() returned empty token")
			}
		})
	}
}

func TestManager_Validate(t *testing.T) {
	manager := NewManager("test-secret-key", time.Hour)
	userID := "user123"

	token, err := manager.Generate(userID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Validate() UserID = %v, want %v", claims.UserID, userID)
	}
}

func TestManager_Validate_InvalidToken(t *testing.T) {
	manager := NewManager("test-secret-key", time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "not-a-jwt-token"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyJ9.wrong_signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.Validate(tt.token)
			if err == nil {
				t.Error("Validate() should fail with invalid token")
			}
		})
	}
}

func TestManager_Validate_WrongSecret(t *testing.T) {
	manager1 := NewManager("secret1", time.Hour)
	manager2 := NewManager("secret2", time.Hour)

	token, err := manager1.Generate("user123")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Пытаемся валидировать токен другим ключом
	_, err = manager2.Validate(token)
	if err == nil {
		t.Error("Validate() should fail with wrong secret")
	}
}

func TestManager_Validate_ExpiredToken(t *testing.T) {
	// Создаём менеджер с очень коротким TTL
	manager := NewManager("test-secret", time.Millisecond)

	token, err := manager.Generate("user123")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Ждём истечения токена
	time.Sleep(10 * time.Millisecond)

	_, err = manager.Validate(token)
	if err == nil {
		t.Error("Validate() should fail with expired token")
	}
}

func TestManager_GetUserID(t *testing.T) {
	manager := NewManager("test-secret-key", time.Hour)
	expectedUserID := "user-456"

	token, err := manager.Generate(expectedUserID)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	userID, err := manager.GetUserID(token)
	if err != nil {
		t.Fatalf("GetUserID() error = %v", err)
	}

	if userID != expectedUserID {
		t.Errorf("GetUserID() = %v, want %v", userID, expectedUserID)
	}
}

func TestManager_GetUserID_InvalidToken(t *testing.T) {
	manager := NewManager("test-secret-key", time.Hour)

	_, err := manager.GetUserID("invalid-token")
	if err == nil {
		t.Error("GetUserID() should fail with invalid token")
	}
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		tokenTTL time.Duration
	}{
		{"normal", "secret", time.Hour},
		{"short ttl", "secret", time.Second},
		{"long ttl", "secret", 24 * time.Hour},
		{"empty secret", "", time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.secret, tt.tokenTTL)
			if manager == nil {
				t.Error("NewManager() returned nil")
			}
		})
	}
}

func TestManager_TokenContainsClaims(t *testing.T) {
	manager := NewManager("test-secret", time.Hour)
	userID := "test-user-id"

	token, _ := manager.Generate(userID)
	claims, _ := manager.Validate(token)

	// Проверяем что claims содержат нужные поля
	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}

	if claims.Issuer != "gophkeeper" {
		t.Errorf("claims.Issuer = %v, want gophkeeper", claims.Issuer)
	}

	if claims.ExpiresAt == nil {
		t.Error("claims.ExpiresAt is nil")
	}

	if claims.IssuedAt == nil {
		t.Error("claims.IssuedAt is nil")
	}
}

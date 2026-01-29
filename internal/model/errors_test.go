package model

import (
	"errors"
	"testing"
)

func TestErrors_NotNil(t *testing.T) {
	// Проверяем что все ошибки определены
	errs := []error{
		ErrUserNotFound,
		ErrUserAlreadyExists,
		ErrInvalidCredentials,
		ErrInvalidToken,
		ErrTokenExpired,
		ErrSecretNotFound,
		ErrSecretAlreadyExists,
		ErrVersionConflict,
		ErrAccessDenied,
		ErrSyncConflict,
	}

	for _, err := range errs {
		if err == nil {
			t.Error("Error should not be nil")
		}
	}
}

func TestErrors_Unique(t *testing.T) {
	errs := []error{
		ErrUserNotFound,
		ErrUserAlreadyExists,
		ErrInvalidCredentials,
		ErrInvalidToken,
		ErrTokenExpired,
		ErrSecretNotFound,
		ErrSecretAlreadyExists,
		ErrVersionConflict,
		ErrAccessDenied,
		ErrSyncConflict,
	}

	// Проверяем уникальность ошибок
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errors.Is(errs[i], errs[j]) {
				t.Errorf("Errors should be unique: %v and %v", errs[i], errs[j])
			}
		}
	}
}

func TestErrors_Messages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"user not found", ErrUserNotFound, "user not found"},
		{"user already exists", ErrUserAlreadyExists, "user already exists"},
		{"invalid credentials", ErrInvalidCredentials, "invalid credentials"},
		{"invalid token", ErrInvalidToken, "invalid or expired token"},
		{"token expired", ErrTokenExpired, "token expired"},
		{"secret not found", ErrSecretNotFound, "secret not found"},
		{"secret already exists", ErrSecretAlreadyExists, "secret with this name already exists"},
		{"version conflict", ErrVersionConflict, "version conflict: secret was modified"},
		{"access denied", ErrAccessDenied, "access denied"},
		{"sync conflict", ErrSyncConflict, "sync conflict"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("Error message = %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}

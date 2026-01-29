package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{"debug level", "debug", false},
		{"info level", "info", false},
		{"warn level", "warn", false},
		{"error level", "error", false},
		{"invalid level", "invalid", true},
		{"empty level", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.level)

			if tt.wantErr {
				if err == nil {
					t.Error("New() should return error for invalid level")
				}
				return
			}

			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if logger == nil {
				t.Error("New() returned nil logger")
			}
		})
	}
}

func TestNewNop(t *testing.T) {
	logger := NewNop()
	if logger == nil {
		t.Error("NewNop() returned nil")
	}

	// Проверяем что можно вызывать методы без паники
	logger.Info("test message")
	logger.Debug("debug message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestLogger_Methods(t *testing.T) {
	logger, err := New("debug")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Проверяем что методы не паникуют
	logger.Info("info message")
	logger.Debug("debug message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Sync не должен возвращать ошибку
	err = logger.Sync()
	// Игнорируем ошибку sync для stdout/stderr (это нормально на некоторых системах)
	_ = err
}

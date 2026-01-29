// Package service содержит бизнес-логику клиента.
package service

import (
	"context"
	"fmt"

	"gophkeeper/internal/client/api"
	"gophkeeper/internal/client/crypto"
	"gophkeeper/internal/client/storage/sqlite"
)

// AuthService сервис аутентификации на клиенте.
type AuthService struct {
	client  *api.Client
	storage *sqlite.DB
}

// NewAuthService создаёт новый сервис аутентификации.
func NewAuthService(client *api.Client, storage *sqlite.DB) *AuthService {
	return &AuthService{
		client:  client,
		storage: storage,
	}
}

// Register регистрирует нового пользователя.
// Создаёт мастер-ключ из пароля и генерирует ключ шифрования данных.
func (s *AuthService) Register(ctx context.Context, login, password, masterPassword string) error {
	// Регистрация на сервере
	resp, err := s.client.Register(ctx, login, password)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Генерируем ключ шифрования данных
	dataKey, err := crypto.GenerateDataKey()
	if err != nil {
		return fmt.Errorf("failed to generate data key: %w", err)
	}

	// Генерируем мастер-ключ из мастер-пароля
	masterKey, salt, err := crypto.DeriveKeyFromPassword(masterPassword, nil)
	if err != nil {
		return fmt.Errorf("failed to derive master key: %w", err)
	}

	// Шифруем Data Key с помощью Master Key
	encryptedDataKey, err := crypto.EncryptDataKey(dataKey, masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt data key: %w", err)
	}

	// Сохраняем соль + зашифрованный ключ
	encryptionKey := append(salt, encryptedDataKey...)

	// Сохраняем сессию локально
	session := &sqlite.Session{
		UserID:        resp.GetUserId(),
		AccessToken:   resp.GetAccessToken(),
		RefreshToken:  resp.GetRefreshToken(),
		EncryptionKey: encryptionKey,
	}

	if err := s.storage.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Устанавливаем токен в клиент
	s.client.SetAccessToken(resp.GetAccessToken())

	return nil
}

// Login выполняет вход пользователя.
func (s *AuthService) Login(ctx context.Context, login, password, masterPassword string) error {
	// Логин на сервере
	resp, err := s.client.Login(ctx, login, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Проверяем, есть ли уже сохранённая сессия с ключом
	existingSession, err := s.storage.GetSession()
	var encryptionKey []byte

	if err == nil && existingSession.UserID == resp.GetUserId() {
		// Используем существующий ключ шифрования
		encryptionKey = existingSession.EncryptionKey
	} else {
		// Новое устройство - нужно создать новый ключ
		dataKey, err := crypto.GenerateDataKey()
		if err != nil {
			return fmt.Errorf("failed to generate data key: %w", err)
		}

		masterKey, salt, err := crypto.DeriveKeyFromPassword(masterPassword, nil)
		if err != nil {
			return fmt.Errorf("failed to derive master key: %w", err)
		}

		encryptedDataKey, err := crypto.EncryptDataKey(dataKey, masterKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt data key: %w", err)
		}

		encryptionKey = append(salt, encryptedDataKey...)
	}

	// Сохраняем сессию
	session := &sqlite.Session{
		UserID:        resp.GetUserId(),
		AccessToken:   resp.GetAccessToken(),
		RefreshToken:  resp.GetRefreshToken(),
		EncryptionKey: encryptionKey,
	}

	if err := s.storage.SaveSession(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	s.client.SetAccessToken(resp.GetAccessToken())

	return nil
}

// Logout выполняет выход пользователя.
func (s *AuthService) Logout(ctx context.Context) error {
	session, err := s.storage.GetSession()
	if err != nil {
		return nil // Уже вышли
	}

	// Инвалидируем токен на сервере
	_ = s.client.Logout(ctx, session.RefreshToken)

	// Удаляем локальную сессию
	if err := s.storage.DeleteSession(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	s.client.SetAccessToken("")

	return nil
}

// RestoreSession восстанавливает сессию из локального хранилища.
func (s *AuthService) RestoreSession(masterPassword string) error {
	session, err := s.storage.GetSession()
	if err != nil {
		return err
	}

	// Проверяем мастер-пароль, пытаясь расшифровать ключ
	_, err = s.GetEncryptor(masterPassword)
	if err != nil {
		return fmt.Errorf("invalid master password: %w", err)
	}

	s.client.SetAccessToken(session.AccessToken)

	return nil
}

// GetEncryptor возвращает шифратор с расшифрованным Data Key.
func (s *AuthService) GetEncryptor(masterPassword string) (*crypto.Encryptor, error) {
	session, err := s.storage.GetSession()
	if err != nil {
		return nil, err
	}

	if len(session.EncryptionKey) < 32 {
		return nil, fmt.Errorf("invalid encryption key")
	}

	// Извлекаем соль (первые 32 байта)
	salt := session.EncryptionKey[:32]
	encryptedDataKey := session.EncryptionKey[32:]

	// Деривируем мастер-ключ
	masterKey, _, err := crypto.DeriveKeyFromPassword(masterPassword, salt)
	if err != nil {
		return nil, err
	}

	// Расшифровываем Data Key
	dataKey, err := crypto.DecryptDataKey(encryptedDataKey, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data key (wrong password?): %w", err)
	}

	return crypto.NewEncryptor(dataKey), nil
}

// RefreshToken обновляет токены.
func (s *AuthService) RefreshToken(ctx context.Context) error {
	session, err := s.storage.GetSession()
	if err != nil {
		return err
	}

	resp, err := s.client.RefreshToken(ctx, session.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	if err := s.storage.UpdateTokens(resp.GetAccessToken(), resp.GetRefreshToken()); err != nil {
		return fmt.Errorf("failed to update tokens: %w", err)
	}

	s.client.SetAccessToken(resp.GetAccessToken())

	return nil
}

// IsLoggedIn проверяет, авторизован ли пользователь.
func (s *AuthService) IsLoggedIn() bool {
	return s.storage.HasSession()
}

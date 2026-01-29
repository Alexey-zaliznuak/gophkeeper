package auth

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"gophkeeper/internal/model"
	"gophkeeper/pkg/crypto"
	"gophkeeper/pkg/jwt"
)

// ==================== Mocks ====================

type mockUserRepo struct {
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	if _, exists := m.users[user.Login]; exists {
		return model.ErrUserAlreadyExists
	}
	user.ID = "user-" + user.Login
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Login] = user
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, model.ErrUserNotFound
}

func (m *mockUserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	if u, exists := m.users[login]; exists {
		return u, nil
	}
	return nil, model.ErrUserNotFound
}

type mockTokenRepo struct {
	tokens map[string]*model.RefreshToken
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{tokens: make(map[string]*model.RefreshToken)}
}

func (m *mockTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	token.ID = "token-" + token.TokenHash[:8]
	token.CreatedAt = time.Now()
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockTokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if t, exists := m.tokens[tokenHash]; exists {
		return t, nil
	}
	return nil, model.ErrInvalidToken
}

func (m *mockTokenRepo) Delete(ctx context.Context, id string) error {
	for hash, t := range m.tokens {
		if t.ID == id {
			delete(m.tokens, hash)
			return nil
		}
	}
	return model.ErrInvalidToken
}

func (m *mockTokenRepo) DeleteByUserID(ctx context.Context, userID string) error {
	for hash, t := range m.tokens {
		if t.UserID == userID {
			delete(m.tokens, hash)
		}
	}
	return nil
}

func (m *mockTokenRepo) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	for hash, t := range m.tokens {
		if t.ExpiresAt.Before(now) {
			delete(m.tokens, hash)
		}
	}
	return nil
}

// ==================== Tests ====================

func setupService() *Service {
	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	jwtManager := jwt.NewManager("test-secret", time.Hour)
	log := zap.NewNop()

	return NewService(userRepo, tokenRepo, jwtManager, 7*24*time.Hour, log)
}

func TestService_Register(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	user, tokens, err := service.Register(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user == nil {
		t.Error("Register() returned nil user")
	}

	if user.Login != "testuser" {
		t.Errorf("user.Login = %v, want testuser", user.Login)
	}

	if tokens == nil {
		t.Error("Register() returned nil tokens")
	}

	if tokens.AccessToken == "" {
		t.Error("AccessToken is empty")
	}

	if tokens.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
}

func TestService_Register_DuplicateUser(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	// Первая регистрация
	_, _, err := service.Register(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// Повторная регистрация
	_, _, err = service.Register(ctx, "testuser", "password456")
	if err != model.ErrUserAlreadyExists {
		t.Errorf("Register() error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestService_Login(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	// Регистрируем
	_, _, _ = service.Register(ctx, "testuser", "password123")

	// Логинимся
	user, tokens, err := service.Login(ctx, "testuser", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if user.Login != "testuser" {
		t.Errorf("user.Login = %v, want testuser", user.Login)
	}

	if tokens.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	_, _, _ = service.Register(ctx, "testuser", "password123")

	_, _, err := service.Login(ctx, "testuser", "wrongpassword")
	if err != model.ErrInvalidCredentials {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_Login_UserNotFound(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	_, _, err := service.Login(ctx, "nonexistent", "password")
	if err != model.ErrInvalidCredentials {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_ValidateAccessToken(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	user, tokens, _ := service.Register(ctx, "testuser", "password123")

	userID, err := service.ValidateAccessToken(ctx, tokens.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if userID != user.ID {
		t.Errorf("userID = %v, want %v", userID, user.ID)
	}
}

func TestService_ValidateAccessToken_Invalid(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	_, err := service.ValidateAccessToken(ctx, "invalid-token")
	if err != model.ErrInvalidToken {
		t.Errorf("ValidateAccessToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestService_RefreshToken(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	_, tokens, _ := service.Register(ctx, "testuser", "password123")

	newTokens, err := service.RefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}

	if newTokens.AccessToken == "" {
		t.Error("new AccessToken is empty")
	}

	if newTokens.RefreshToken == "" {
		t.Error("new RefreshToken is empty")
	}

	// Старый токен должен быть инвалидирован
	_, err = service.RefreshToken(ctx, tokens.RefreshToken)
	if err == nil {
		t.Error("Old refresh token should be invalid")
	}
}

func TestService_RefreshToken_Invalid(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	_, err := service.RefreshToken(ctx, "invalid-refresh-token")
	if err != model.ErrInvalidToken {
		t.Errorf("RefreshToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestService_Logout(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	_, tokens, _ := service.Register(ctx, "testuser", "password123")

	err := service.Logout(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	// Токен должен быть инвалидирован
	_, err = service.RefreshToken(ctx, tokens.RefreshToken)
	if err == nil {
		t.Error("Refresh token should be invalid after logout")
	}
}

func TestService_Logout_InvalidToken(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	// Logout с невалидным токеном не должен возвращать ошибку
	err := service.Logout(ctx, "invalid-token")
	if err != nil {
		t.Errorf("Logout() error = %v, want nil", err)
	}
}

func TestService_PasswordHashed(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	user, _, _ := service.Register(ctx, "testuser", "password123")

	// Пароль должен быть хеширован
	if user.PasswordHash == "password123" {
		t.Error("Password should be hashed")
	}

	// Но должен проверяться корректно
	if !crypto.CheckPassword("password123", user.PasswordHash) {
		t.Error("Password hash should be valid")
	}
}

func TestService_IsLoggedIn(t *testing.T) {
	service := setupService()

	// IsLoggedIn не реализован в серверном сервисе (это клиентская функция)
	// Проверяем что сервис создаётся корректно
	if service == nil {
		t.Error("Service is nil")
	}
}

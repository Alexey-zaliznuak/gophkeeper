package secret

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"gophkeeper/internal/model"
)

// ==================== Mocks ====================

type mockSecretRepo struct {
	secrets map[string]*model.Secret
}

func newMockSecretRepo() *mockSecretRepo {
	return &mockSecretRepo{secrets: make(map[string]*model.Secret)}
}

func (m *mockSecretRepo) Create(ctx context.Context, secret *model.Secret) error {
	// Проверяем уникальность имени для пользователя
	for _, s := range m.secrets {
		if s.UserID == secret.UserID && s.Name == secret.Name && s.DeletedAt == nil {
			return model.ErrSecretAlreadyExists
		}
	}

	secret.ID = "secret-" + secret.Name
	secret.Version = 1
	secret.CreatedAt = time.Now()
	secret.UpdatedAt = time.Now()
	m.secrets[secret.ID] = secret
	return nil
}

func (m *mockSecretRepo) Update(ctx context.Context, secret *model.Secret) error {
	existing, exists := m.secrets[secret.ID]
	if !exists || existing.DeletedAt != nil {
		return model.ErrSecretNotFound
	}

	// Проверяем оптимистичную блокировку: ожидаемая версия должна совпадать с текущей
	if existing.Version != secret.Version {
		return model.ErrVersionConflict
	}

	// Увеличиваем версию и обновляем
	newSecret := *secret
	newSecret.Version = existing.Version + 1
	newSecret.UpdatedAt = time.Now()
	m.secrets[secret.ID] = &newSecret

	// Обновляем переданный secret
	secret.Version = newSecret.Version
	secret.UpdatedAt = newSecret.UpdatedAt

	return nil
}

func (m *mockSecretRepo) Delete(ctx context.Context, userID, secretID string) error {
	secret, exists := m.secrets[secretID]
	if !exists || secret.DeletedAt != nil {
		return model.ErrSecretNotFound
	}

	if secret.UserID != userID {
		return model.ErrAccessDenied
	}

	now := time.Now()
	secret.DeletedAt = &now
	return nil
}

func (m *mockSecretRepo) GetByID(ctx context.Context, userID, secretID string) (*model.Secret, error) {
	secret, exists := m.secrets[secretID]
	if !exists || secret.DeletedAt != nil {
		return nil, model.ErrSecretNotFound
	}

	if secret.UserID != userID {
		return nil, model.ErrAccessDenied
	}

	return secret, nil
}

func (m *mockSecretRepo) GetByName(ctx context.Context, userID, name string) (*model.Secret, error) {
	for _, s := range m.secrets {
		if s.UserID == userID && s.Name == name && s.DeletedAt == nil {
			return s, nil
		}
	}
	return nil, model.ErrSecretNotFound
}

func (m *mockSecretRepo) List(ctx context.Context, userID string) ([]*model.Secret, error) {
	var result []*model.Secret
	for _, s := range m.secrets {
		if s.UserID == userID && s.DeletedAt == nil {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSecretRepo) ListChangedSince(ctx context.Context, userID string, since time.Time) ([]*model.Secret, error) {
	var result []*model.Secret
	for _, s := range m.secrets {
		if s.UserID == userID && s.UpdatedAt.After(since) && s.DeletedAt == nil {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSecretRepo) ListDeletedSince(ctx context.Context, userID string, since time.Time) ([]string, error) {
	var result []string
	for _, s := range m.secrets {
		if s.UserID == userID && s.DeletedAt != nil && s.DeletedAt.After(since) {
			result = append(result, s.ID)
		}
	}
	return result, nil
}

// ==================== Tests ====================

func setupService() *Service {
	repo := newMockSecretRepo()
	log := zap.NewNop()
	return NewService(repo, log)
}

func TestService_Create(t *testing.T) {
	service := setupService()
	ctx := context.Background()
	userID := "user-123"

	secret := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeCredentials,
		EncryptedData: []byte("encrypted-data"),
		Metadata:      map[string]string{"site": "example.com"},
	}

	created, err := service.Create(ctx, userID, secret)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID == "" {
		t.Error("Created secret should have ID")
	}

	if created.UserID != userID {
		t.Errorf("UserID = %v, want %v", created.UserID, userID)
	}

	if created.Version != 1 {
		t.Errorf("Version = %v, want 1", created.Version)
	}
}

func TestService_Create_Duplicate(t *testing.T) {
	service := setupService()
	ctx := context.Background()
	userID := "user-123"

	secret := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
	}

	_, _ = service.Create(ctx, userID, secret)

	// Попытка создать с тем же именем
	secret2 := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("other data"),
	}

	_, err := service.Create(ctx, userID, secret2)
	if err != model.ErrSecretAlreadyExists {
		t.Errorf("Create() error = %v, want ErrSecretAlreadyExists", err)
	}
}

func TestService_Get(t *testing.T) {
	service := setupService()
	ctx := context.Background()
	userID := "user-123"

	secret := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeCredentials,
		EncryptedData: []byte("data"),
	}
	created, _ := service.Create(ctx, userID, secret)

	got, err := service.Get(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Name != secret.Name {
		t.Errorf("Name = %v, want %v", got.Name, secret.Name)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	_, err := service.Get(ctx, "user-123", "non-existent")
	if err != model.ErrSecretNotFound {
		t.Errorf("Get() error = %v, want ErrSecretNotFound", err)
	}
}

func TestService_Get_WrongUser(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	secret := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
	}
	created, _ := service.Create(ctx, "user-1", secret)

	// Пытаемся получить от другого пользователя
	_, err := service.Get(ctx, "user-2", created.ID)
	if err != model.ErrAccessDenied {
		t.Errorf("Get() error = %v, want ErrAccessDenied", err)
	}
}

func TestService_Update(t *testing.T) {
	service := setupService()
	ctx := context.Background()
	userID := "user-123"

	secret := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("original"),
	}
	created, _ := service.Create(ctx, userID, secret)

	// Обновляем
	created.Name = "updated-name"
	created.EncryptedData = []byte("updated")

	updated, err := service.Update(ctx, userID, created, 1)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "updated-name" {
		t.Errorf("Name = %v, want updated-name", updated.Name)
	}

	if updated.Version != 2 {
		t.Errorf("Version = %v, want 2", updated.Version)
	}
}

func TestService_Update_VersionConflict(t *testing.T) {
	service := setupService()
	ctx := context.Background()
	userID := "user-123"

	secret := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
	}
	created, _ := service.Create(ctx, userID, secret)

	// Первое обновление - должно пройти
	created.Name = "updated-name"
	updated, _ := service.Update(ctx, userID, created, 1)

	// Пытаемся обновить с устаревшей версией (1 вместо 2)
	updated.Name = "another-name"
	_, err := service.Update(ctx, userID, updated, 1) // версия 1 уже устарела
	if err != model.ErrVersionConflict {
		t.Errorf("Update() error = %v, want ErrVersionConflict", err)
	}
}

func TestService_Delete(t *testing.T) {
	service := setupService()
	ctx := context.Background()
	userID := "user-123"

	secret := &model.Secret{
		Name:          "test-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
	}
	created, _ := service.Create(ctx, userID, secret)

	err := service.Delete(ctx, userID, created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Попытка получить удалённый секрет
	_, err = service.Get(ctx, userID, created.ID)
	if err != model.ErrSecretNotFound {
		t.Errorf("Get() error = %v, want ErrSecretNotFound", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	err := service.Delete(ctx, "user-123", "non-existent")
	if err != model.ErrSecretNotFound {
		t.Errorf("Delete() error = %v, want ErrSecretNotFound", err)
	}
}

func TestService_List(t *testing.T) {
	service := setupService()
	ctx := context.Background()
	userID := "user-123"

	// Создаём несколько секретов
	for i := 0; i < 3; i++ {
		secret := &model.Secret{
			Name:          "secret-" + string(rune('A'+i)),
			Type:          model.SecretTypeText,
			EncryptedData: []byte("data"),
		}
		service.Create(ctx, userID, secret)
	}

	// Создаём секрет для другого пользователя
	otherSecret := &model.Secret{
		Name:          "other-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
	}
	service.Create(ctx, "other-user", otherSecret)

	secrets, err := service.List(ctx, userID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(secrets) != 3 {
		t.Errorf("List() returned %d secrets, want 3", len(secrets))
	}
}

func TestService_List_Empty(t *testing.T) {
	service := setupService()
	ctx := context.Background()

	secrets, err := service.List(ctx, "user-with-no-secrets")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(secrets) != 0 {
		t.Errorf("List() returned %d secrets, want 0", len(secrets))
	}
}

package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"gophkeeper/internal/model"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "gophkeeper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	db, err := New(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create DB: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestNew(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "gophkeeper-test-*")
	defer os.RemoveAll(tmpDir)

	db, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// Проверяем что файл создан
	dbPath := filepath.Join(tmpDir, "gophkeeper.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestNew_CreatesDirectory(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "gophkeeper-test-*")
	defer os.RemoveAll(tmpDir)

	nestedDir := filepath.Join(tmpDir, "nested", "dir")

	db, err := New(nestedDir)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

func TestDB_Close(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestDB_GetDB(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	sqlDB := db.GetDB()
	if sqlDB == nil {
		t.Error("GetDB() returned nil")
	}

	// Проверяем что можно выполнить запрос
	err := sqlDB.Ping()
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

// ==================== Session Tests ====================

func TestDB_Session(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	session := &Session{
		UserID:        "user-123",
		AccessToken:   "access-token",
		RefreshToken:  "refresh-token",
		EncryptionKey: []byte("encryption-key-32-bytes-long!!!!"),
	}

	// Сохраняем сессию
	if err := db.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Получаем сессию
	got, err := db.GetSession()
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if got.UserID != session.UserID {
		t.Errorf("UserID = %v, want %v", got.UserID, session.UserID)
	}
	if got.AccessToken != session.AccessToken {
		t.Errorf("AccessToken = %v, want %v", got.AccessToken, session.AccessToken)
	}
	if got.RefreshToken != session.RefreshToken {
		t.Errorf("RefreshToken = %v, want %v", got.RefreshToken, session.RefreshToken)
	}
}

func TestDB_GetSession_NoSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := db.GetSession()
	if err != ErrNoSession {
		t.Errorf("GetSession() error = %v, want ErrNoSession", err)
	}
}

func TestDB_UpdateTokens(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Сначала сохраняем сессию
	session := &Session{
		UserID:        "user-123",
		AccessToken:   "old-access",
		RefreshToken:  "old-refresh",
		EncryptionKey: []byte("key"),
	}
	db.SaveSession(session)

	// Обновляем токены
	err := db.UpdateTokens("new-access", "new-refresh")
	if err != nil {
		t.Fatalf("UpdateTokens() error = %v", err)
	}

	// Проверяем
	got, _ := db.GetSession()
	if got.AccessToken != "new-access" {
		t.Errorf("AccessToken = %v, want new-access", got.AccessToken)
	}
	if got.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %v, want new-refresh", got.RefreshToken)
	}
}

func TestDB_UpdateTokens_NoSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.UpdateTokens("access", "refresh")
	if err != ErrNoSession {
		t.Errorf("UpdateTokens() error = %v, want ErrNoSession", err)
	}
}

func TestDB_DeleteSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	session := &Session{
		UserID:        "user-123",
		AccessToken:   "access",
		RefreshToken:  "refresh",
		EncryptionKey: []byte("key"),
	}
	db.SaveSession(session)

	if err := db.DeleteSession(); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	if db.HasSession() {
		t.Error("HasSession() = true after DeleteSession()")
	}
}

func TestDB_HasSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db.HasSession() {
		t.Error("HasSession() = true on empty DB")
	}

	session := &Session{
		UserID:        "user",
		AccessToken:   "access",
		RefreshToken:  "refresh",
		EncryptionKey: []byte("key"),
	}
	db.SaveSession(session)

	if !db.HasSession() {
		t.Error("HasSession() = false after SaveSession()")
	}
}

// ==================== Secret Tests ====================

func TestDB_CreateSecret(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	secret := &LocalSecret{
		Name:          "test-secret",
		Type:          model.SecretTypeCredentials,
		EncryptedData: []byte("encrypted"),
		Metadata:      map[string]string{"key": "value"},
	}

	if err := db.CreateSecret(secret); err != nil {
		t.Fatalf("CreateSecret() error = %v", err)
	}

	if secret.ID == "" {
		t.Error("CreateSecret() did not set ID")
	}
}

func TestDB_GetSecretByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	secret := &LocalSecret{
		Name:          "test-secret",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
		Metadata:      map[string]string{},
	}
	db.CreateSecret(secret)

	got, err := db.GetSecretByID(secret.ID)
	if err != nil {
		t.Fatalf("GetSecretByID() error = %v", err)
	}

	if got.Name != secret.Name {
		t.Errorf("Name = %v, want %v", got.Name, secret.Name)
	}
}

func TestDB_GetSecretByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := db.GetSecretByID("non-existent")
	if err != ErrSecretNotFound {
		t.Errorf("GetSecretByID() error = %v, want ErrSecretNotFound", err)
	}
}

func TestDB_GetSecretByName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	secret := &LocalSecret{
		Name:          "my-secret",
		Type:          model.SecretTypeCredentials,
		EncryptedData: []byte("data"),
		Metadata:      map[string]string{},
	}
	db.CreateSecret(secret)

	got, err := db.GetSecretByName("my-secret")
	if err != nil {
		t.Fatalf("GetSecretByName() error = %v", err)
	}

	if got.ID != secret.ID {
		t.Errorf("ID = %v, want %v", got.ID, secret.ID)
	}
}

func TestDB_ListSecrets(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём несколько секретов
	for i := 0; i < 3; i++ {
		secret := &LocalSecret{
			Name:          "secret-" + string(rune('A'+i)),
			Type:          model.SecretTypeText,
			EncryptedData: []byte("data"),
			Metadata:      map[string]string{},
		}
		db.CreateSecret(secret)
	}

	secrets, err := db.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(secrets) != 3 {
		t.Errorf("ListSecrets() returned %d secrets, want 3", len(secrets))
	}
}

func TestDB_ListSecretsByType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Создаём секреты разных типов
	types := []model.SecretType{
		model.SecretTypeCredentials,
		model.SecretTypeCredentials,
		model.SecretTypeText,
		model.SecretTypeCard,
	}

	for i, typ := range types {
		secret := &LocalSecret{
			Name:          "secret-" + string(rune('A'+i)),
			Type:          typ,
			EncryptedData: []byte("data"),
			Metadata:      map[string]string{},
		}
		db.CreateSecret(secret)
	}

	credentials, _ := db.ListSecretsByType(model.SecretTypeCredentials)
	if len(credentials) != 2 {
		t.Errorf("ListSecretsByType(Credentials) = %d, want 2", len(credentials))
	}

	text, _ := db.ListSecretsByType(model.SecretTypeText)
	if len(text) != 1 {
		t.Errorf("ListSecretsByType(Text) = %d, want 1", len(text))
	}
}

func TestDB_UpdateSecret(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	secret := &LocalSecret{
		Name:          "original-name",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("original"),
		Metadata:      map[string]string{},
	}
	db.CreateSecret(secret)

	// Обновляем
	secret.Name = "updated-name"
	secret.EncryptedData = []byte("updated")

	if err := db.UpdateSecret(secret); err != nil {
		t.Fatalf("UpdateSecret() error = %v", err)
	}

	// Проверяем
	got, _ := db.GetSecretByID(secret.ID)
	if got.Name != "updated-name" {
		t.Errorf("Name = %v, want updated-name", got.Name)
	}
}

func TestDB_DeleteSecret(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	secret := &LocalSecret{
		Name:          "to-delete",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
		Metadata:      map[string]string{},
	}
	db.CreateSecret(secret)

	if err := db.DeleteSecret(secret.ID); err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	// Проверяем что секрет помечен как удалённый
	_, err := db.GetSecretByID(secret.ID)
	if err != ErrSecretNotFound {
		t.Error("Deleted secret should not be found")
	}
}

func TestDB_DeleteSecret_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.DeleteSecret("non-existent")
	if err != ErrSecretNotFound {
		t.Errorf("DeleteSecret() error = %v, want ErrSecretNotFound", err)
	}
}

// ==================== Sync Tests ====================

func TestDB_SyncMeta(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Изначально nil
	lastSync, err := db.GetLastSyncTime()
	if err != nil {
		t.Fatalf("GetLastSyncTime() error = %v", err)
	}
	if lastSync != nil {
		t.Error("GetLastSyncTime() should return nil for new DB")
	}
}

func TestDB_MarkAsSynced(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	secret := &LocalSecret{
		Name:          "unsynced",
		Type:          model.SecretTypeText,
		EncryptedData: []byte("data"),
		Metadata:      map[string]string{},
		IsSynced:      false,
	}
	db.CreateSecret(secret)

	if err := db.MarkAsSynced(secret.ID, "server-id-123"); err != nil {
		t.Fatalf("MarkAsSynced() error = %v", err)
	}

	got, _ := db.GetSecretByID(secret.ID)
	if !got.IsSynced {
		t.Error("Secret should be marked as synced")
	}
	if got.ServerID != "server-id-123" {
		t.Errorf("ServerID = %v, want server-id-123", got.ServerID)
	}
}

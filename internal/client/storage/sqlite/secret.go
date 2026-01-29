package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gophkeeper/internal/model"
)

// LocalSecret представляет секрет в локальном хранилище.
type LocalSecret struct {
	ID            string
	ServerID      string
	Name          string
	Type          model.SecretType
	EncryptedData []byte
	Metadata      map[string]string
	Version       int64
	IsSynced      bool
	IsDeleted     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	SyncedAt      *time.Time
}

// ErrSecretNotFound возвращается когда секрет не найден.
var ErrSecretNotFound = errors.New("secret not found")

// CreateSecret создаёт новый секрет локально.
func (d *DB) CreateSecret(secret *LocalSecret) error {
	if secret.ID == "" {
		secret.ID = uuid.New().String()
	}

	metadata, err := json.Marshal(secret.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO secrets (id, server_id, name, type, encrypted_data, metadata, version, is_synced, is_deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.Exec(query,
		secret.ID,
		secret.ServerID,
		secret.Name,
		secret.Type,
		secret.EncryptedData,
		string(metadata),
		secret.Version,
		secret.IsSynced,
		secret.IsDeleted,
	)

	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	// Добавляем в очередь синхронизации
	if !secret.IsSynced {
		if err := d.addToSyncQueue(secret.ID, "create"); err != nil {
			return err
		}
	}

	return nil
}

// UpdateSecret обновляет секрет.
func (d *DB) UpdateSecret(secret *LocalSecret) error {
	metadata, err := json.Marshal(secret.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE secrets
		SET name = ?, encrypted_data = ?, metadata = ?, version = version + 1, 
		    is_synced = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_deleted = 0
	`

	result, err := d.db.Exec(query, secret.Name, secret.EncryptedData, string(metadata), secret.ID)
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrSecretNotFound
	}

	// Добавляем в очередь синхронизации
	return d.addToSyncQueue(secret.ID, "update")
}

// DeleteSecret помечает секрет как удалённый.
func (d *DB) DeleteSecret(id string) error {
	query := `
		UPDATE secrets
		SET is_deleted = 1, is_synced = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_deleted = 0
	`

	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrSecretNotFound
	}

	return d.addToSyncQueue(id, "delete")
}

// GetSecretByID возвращает секрет по ID.
func (d *DB) GetSecretByID(id string) (*LocalSecret, error) {
	query := `
		SELECT id, server_id, name, type, encrypted_data, metadata, version, 
		       is_synced, is_deleted, created_at, updated_at, synced_at
		FROM secrets
		WHERE id = ? AND is_deleted = 0
	`

	return d.scanSecret(d.db.QueryRow(query, id))
}

// GetSecretByName возвращает секрет по имени.
func (d *DB) GetSecretByName(name string) (*LocalSecret, error) {
	query := `
		SELECT id, server_id, name, type, encrypted_data, metadata, version, 
		       is_synced, is_deleted, created_at, updated_at, synced_at
		FROM secrets
		WHERE name = ? AND is_deleted = 0
	`

	return d.scanSecret(d.db.QueryRow(query, name))
}

// ListSecrets возвращает все секреты.
func (d *DB) ListSecrets() ([]*LocalSecret, error) {
	query := `
		SELECT id, server_id, name, type, encrypted_data, metadata, version, 
		       is_synced, is_deleted, created_at, updated_at, synced_at
		FROM secrets
		WHERE is_deleted = 0
		ORDER BY name
	`

	return d.scanSecrets(query)
}

// ListSecretsByType возвращает секреты по типу.
func (d *DB) ListSecretsByType(secretType model.SecretType) ([]*LocalSecret, error) {
	query := `
		SELECT id, server_id, name, type, encrypted_data, metadata, version, 
		       is_synced, is_deleted, created_at, updated_at, synced_at
		FROM secrets
		WHERE type = ? AND is_deleted = 0
		ORDER BY name
	`

	rows, err := d.db.Query(query, secretType)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer rows.Close()

	return d.scanSecretsRows(rows)
}

// GetUnsyncedSecrets возвращает несинхронизированные секреты.
func (d *DB) GetUnsyncedSecrets() ([]*LocalSecret, error) {
	query := `
		SELECT id, server_id, name, type, encrypted_data, metadata, version, 
		       is_synced, is_deleted, created_at, updated_at, synced_at
		FROM secrets
		WHERE is_synced = 0
		ORDER BY updated_at
	`

	return d.scanSecrets(query)
}

// MarkAsSynced помечает секрет как синхронизированный.
func (d *DB) MarkAsSynced(id, serverID string) error {
	query := `
		UPDATE secrets
		SET is_synced = 1, server_id = ?, synced_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := d.db.Exec(query, serverID, id)
	return err
}

// UpsertFromServer вставляет или обновляет секрет с сервера.
func (d *DB) UpsertFromServer(secret *LocalSecret) error {
	metadata, err := json.Marshal(secret.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO secrets (id, server_id, name, type, encrypted_data, metadata, version, is_synced, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			encrypted_data = excluded.encrypted_data,
			metadata = excluded.metadata,
			version = excluded.version,
			is_synced = 1,
			synced_at = CURRENT_TIMESTAMP
		WHERE secrets.version < excluded.version
	`

	_, err = d.db.Exec(query,
		secret.ID,
		secret.ServerID,
		secret.Name,
		secret.Type,
		secret.EncryptedData,
		string(metadata),
		secret.Version,
	)

	return err
}

// DeleteByServerID удаляет секрет по серверному ID (при синхронизации удалений).
func (d *DB) DeleteByServerID(serverID string) error {
	query := `DELETE FROM secrets WHERE server_id = ?`
	_, err := d.db.Exec(query, serverID)
	return err
}

// ==================== Sync Queue ====================

// SyncQueueItem элемент очереди синхронизации.
type SyncQueueItem struct {
	ID        int64
	SecretID  string
	Operation string
	CreatedAt time.Time
}

// addToSyncQueue добавляет операцию в очередь синхронизации.
func (d *DB) addToSyncQueue(secretID, operation string) error {
	query := `INSERT INTO sync_queue (secret_id, operation) VALUES (?, ?)`
	_, err := d.db.Exec(query, secretID, operation)
	return err
}

// GetSyncQueue возвращает очередь синхронизации.
func (d *DB) GetSyncQueue() ([]*SyncQueueItem, error) {
	query := `SELECT id, secret_id, operation, created_at FROM sync_queue ORDER BY created_at`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*SyncQueueItem
	for rows.Next() {
		item := &SyncQueueItem{}
		if err := rows.Scan(&item.ID, &item.SecretID, &item.Operation, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// ClearSyncQueue очищает очередь синхронизации.
func (d *DB) ClearSyncQueue() error {
	_, err := d.db.Exec("DELETE FROM sync_queue")
	return err
}

// RemoveFromSyncQueue удаляет элемент из очереди.
func (d *DB) RemoveFromSyncQueue(id int64) error {
	_, err := d.db.Exec("DELETE FROM sync_queue WHERE id = ?", id)
	return err
}

// ==================== Sync Meta ====================

// GetLastSyncTime возвращает время последней синхронизации.
func (d *DB) GetLastSyncTime() (*time.Time, error) {
	var syncTime sql.NullTime
	err := d.db.QueryRow("SELECT last_sync_time FROM sync_meta WHERE id = 1").Scan(&syncTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if !syncTime.Valid {
		return nil, nil
	}

	return &syncTime.Time, nil
}

// SetLastSyncTime устанавливает время последней синхронизации.
func (d *DB) SetLastSyncTime(t time.Time) error {
	query := `INSERT OR REPLACE INTO sync_meta (id, last_sync_time) VALUES (1, ?)`
	_, err := d.db.Exec(query, t)
	return err
}

// ==================== Helpers ====================

func (d *DB) scanSecret(row *sql.Row) (*LocalSecret, error) {
	secret := &LocalSecret{}
	var metadataStr string
	var isSynced, isDeleted int
	var syncedAt sql.NullTime
	var serverID sql.NullString

	err := row.Scan(
		&secret.ID,
		&serverID,
		&secret.Name,
		&secret.Type,
		&secret.EncryptedData,
		&metadataStr,
		&secret.Version,
		&isSynced,
		&isDeleted,
		&secret.CreatedAt,
		&secret.UpdatedAt,
		&syncedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSecretNotFound
		}
		return nil, fmt.Errorf("failed to scan secret: %w", err)
	}

	if serverID.Valid {
		secret.ServerID = serverID.String
	}

	if err := json.Unmarshal([]byte(metadataStr), &secret.Metadata); err != nil {
		secret.Metadata = make(map[string]string)
	}

	secret.IsSynced = isSynced == 1
	secret.IsDeleted = isDeleted == 1
	if syncedAt.Valid {
		secret.SyncedAt = &syncedAt.Time
	}

	return secret, nil
}

func (d *DB) scanSecrets(query string) ([]*LocalSecret, error) {
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query secrets: %w", err)
	}
	defer rows.Close()

	return d.scanSecretsRows(rows)
}

func (d *DB) scanSecretsRows(rows *sql.Rows) ([]*LocalSecret, error) {
	var secrets []*LocalSecret

	for rows.Next() {
		secret := &LocalSecret{}
		var metadataStr string
		var isSynced, isDeleted int
		var syncedAt sql.NullTime
		var serverID sql.NullString

		err := rows.Scan(
			&secret.ID,
			&serverID,
			&secret.Name,
			&secret.Type,
			&secret.EncryptedData,
			&metadataStr,
			&secret.Version,
			&isSynced,
			&isDeleted,
			&secret.CreatedAt,
			&secret.UpdatedAt,
			&syncedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan secret: %w", err)
		}

		if serverID.Valid {
			secret.ServerID = serverID.String
		}

		if err := json.Unmarshal([]byte(metadataStr), &secret.Metadata); err != nil {
			secret.Metadata = make(map[string]string)
		}

		secret.IsSynced = isSynced == 1
		secret.IsDeleted = isDeleted == 1
		if syncedAt.Valid {
			secret.SyncedAt = &syncedAt.Time
		}

		secrets = append(secrets, secret)
	}

	return secrets, rows.Err()
}

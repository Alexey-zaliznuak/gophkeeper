package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"gophkeeper/internal/model"
)

// SecretRepository реализует repository.SecretRepository для PostgreSQL.
type SecretRepository struct {
	db *DB
}

// NewSecretRepository создаёт новый репозиторий секретов.
func NewSecretRepository(db *DB) *SecretRepository {
	return &SecretRepository{db: db}
}

// Create создаёт новый секрет.
func (r *SecretRepository) Create(ctx context.Context, secret *model.Secret) error {
	metadata, err := json.Marshal(secret.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO secrets (user_id, name, type, encrypted_data, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, version, created_at, updated_at
	`

	err = r.db.Pool.QueryRow(ctx, query,
		secret.UserID,
		secret.Name,
		secret.Type,
		secret.EncryptedData,
		metadata,
	).Scan(&secret.ID, &secret.Version, &secret.CreatedAt, &secret.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return model.ErrSecretAlreadyExists
		}
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// Update обновляет существующий секрет с проверкой версии.
func (r *SecretRepository) Update(ctx context.Context, secret *model.Secret) error {
	metadata, err := json.Marshal(secret.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE secrets
		SET name = $1, encrypted_data = $2, metadata = $3, version = version + 1
		WHERE id = $4 AND user_id = $5 AND version = $6 AND deleted_at IS NULL
		RETURNING version, updated_at
	`

	err = r.db.Pool.QueryRow(ctx, query,
		secret.Name,
		secret.EncryptedData,
		metadata,
		secret.ID,
		secret.UserID,
		secret.Version,
	).Scan(&secret.Version, &secret.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Проверяем, существует ли секрет вообще
			exists, _ := r.existsForUser(ctx, secret.UserID, secret.ID)
			if !exists {
				return model.ErrSecretNotFound
			}
			return model.ErrVersionConflict
		}
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

// Delete выполняет soft delete секрета.
func (r *SecretRepository) Delete(ctx context.Context, userID, secretID string) error {
	query := `
		UPDATE secrets
		SET deleted_at = NOW(), version = version + 1
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.Pool.Exec(ctx, query, secretID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrSecretNotFound
	}

	return nil
}

// GetByID возвращает секрет по ID.
func (r *SecretRepository) GetByID(ctx context.Context, userID, secretID string) (*model.Secret, error) {
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	return r.scanSecret(ctx, query, secretID, userID)
}

// GetByName возвращает секрет по имени.
func (r *SecretRepository) GetByName(ctx context.Context, userID, name string) (*model.Secret, error) {
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE name = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	return r.scanSecret(ctx, query, name, userID)
}

// List возвращает все активные секреты пользователя.
func (r *SecretRepository) List(ctx context.Context, userID string) ([]*model.Secret, error) {
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY name
	`

	return r.scanSecrets(ctx, query, userID)
}

// ListChangedSince возвращает секреты, изменённые после указанного времени.
func (r *SecretRepository) ListChangedSince(ctx context.Context, userID string, since time.Time) ([]*model.Secret, error) {
	query := `
		SELECT id, user_id, name, type, encrypted_data, metadata, version, created_at, updated_at, deleted_at
		FROM secrets
		WHERE user_id = $1 AND updated_at > $2 AND deleted_at IS NULL
		ORDER BY updated_at
	`

	return r.scanSecrets(ctx, query, userID, since)
}

// ListDeletedSince возвращает ID удалённых секретов после указанного времени.
func (r *SecretRepository) ListDeletedSince(ctx context.Context, userID string, since time.Time) ([]string, error) {
	query := `
		SELECT id
		FROM secrets
		WHERE user_id = $1 AND deleted_at > $2
		ORDER BY deleted_at
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to list deleted secrets: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan deleted secret id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// scanSecret сканирует один секрет из результата запроса.
func (r *SecretRepository) scanSecret(ctx context.Context, query string, args ...interface{}) (*model.Secret, error) {
	secret := &model.Secret{}
	var metadataJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, args...).Scan(
		&secret.ID,
		&secret.UserID,
		&secret.Name,
		&secret.Type,
		&secret.EncryptedData,
		&metadataJSON,
		&secret.Version,
		&secret.CreatedAt,
		&secret.UpdatedAt,
		&secret.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrSecretNotFound
		}
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &secret.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return secret, nil
}

// scanSecrets сканирует список секретов из результата запроса.
func (r *SecretRepository) scanSecrets(ctx context.Context, query string, args ...interface{}) ([]*model.Secret, error) {
	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*model.Secret
	for rows.Next() {
		secret := &model.Secret{}
		var metadataJSON []byte

		err := rows.Scan(
			&secret.ID,
			&secret.UserID,
			&secret.Name,
			&secret.Type,
			&secret.EncryptedData,
			&metadataJSON,
			&secret.Version,
			&secret.CreatedAt,
			&secret.UpdatedAt,
			&secret.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan secret: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &secret.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		secrets = append(secrets, secret)
	}

	return secrets, rows.Err()
}

// existsForUser проверяет, существует ли секрет для пользователя.
func (r *SecretRepository) existsForUser(ctx context.Context, userID, secretID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM secrets WHERE id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, secretID, userID).Scan(&exists)
	return exists, err
}

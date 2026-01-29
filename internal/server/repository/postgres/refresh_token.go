package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"gophkeeper/internal/model"
)

// RefreshTokenRepository реализует repository.RefreshTokenRepository для PostgreSQL.
type RefreshTokenRepository struct {
	db *DB
}

// NewRefreshTokenRepository создаёт новый репозиторий refresh токенов.
func NewRefreshTokenRepository(db *DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create создаёт новый refresh токен.
func (r *RefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetByTokenHash возвращает токен по хешу.
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	token := &model.RefreshToken{}
	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

// Delete удаляет токен по ID.
func (r *RefreshTokenRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrInvalidToken
	}

	return nil
}

// DeleteByUserID удаляет все токены пользователя.
func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`

	_, err := r.db.Pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user refresh tokens: %w", err)
	}

	return nil
}

// DeleteExpired удаляет все истёкшие токены.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW()`

	_, err := r.db.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}

	return nil
}

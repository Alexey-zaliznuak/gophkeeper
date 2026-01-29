package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"gophkeeper/internal/model"
)

// UserRepository реализует repository.UserRepository для PostgreSQL.
type UserRepository struct {
	db *DB
}

// NewUserRepository создаёт новый репозиторий пользователей.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create создаёт нового пользователя.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query, user.Login, user.PasswordHash).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		// Проверка на дубликат
		if isDuplicateKeyError(err) {
			return model.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID возвращает пользователя по ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, login, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &model.User{}
	err := r.db.Pool.QueryRow(ctx, query, id).
		Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// GetByLogin возвращает пользователя по логину.
func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	query := `
		SELECT id, login, password_hash, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	user := &model.User{}
	err := r.db.Pool.QueryRow(ctx, query, login).
		Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	return user, nil
}

// isDuplicateKeyError проверяет, является ли ошибка нарушением уникальности.
func isDuplicateKeyError(err error) bool {
	// PostgreSQL error code 23505 = unique_violation
	return err != nil && contains(err.Error(), "23505")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

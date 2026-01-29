// Package postgres содержит реализацию репозиториев для PostgreSQL.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB обёртка над пулом соединений PostgreSQL.
type DB struct {
	Pool *pgxpool.Pool
}

// New создаёт новое подключение к PostgreSQL.
func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close закрывает пул соединений.
func (db *DB) Close() {
	db.Pool.Close()
}

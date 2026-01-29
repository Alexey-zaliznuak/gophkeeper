// Package sqlite содержит локальное хранилище на базе SQLite.
package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB обёртка над SQLite базой данных.
type DB struct {
	db *sql.DB
}

// New создаёт новое подключение к SQLite.
func New(dataDir string) (*DB, error) {
	// Создаём директорию если не существует
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "gophkeeper.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Инициализируем схему
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return &DB{db: db}, nil
}

// Close закрывает соединение с базой данных.
func (d *DB) Close() error {
	return d.db.Close()
}

// GetDB возвращает внутренний объект базы данных.
func (d *DB) GetDB() *sql.DB {
	return d.db
}

// initSchema создаёт таблицы.
func initSchema(db *sql.DB) error {
	schema := `
	-- Таблица сессии (хранит токены и данные пользователя)
	CREATE TABLE IF NOT EXISTS session (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		user_id TEXT NOT NULL,
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		encryption_key BLOB NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Таблица секретов (локальный кеш)
	CREATE TABLE IF NOT EXISTS secrets (
		id TEXT PRIMARY KEY,
		server_id TEXT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		encrypted_data BLOB NOT NULL,
		metadata TEXT DEFAULT '{}',
		version INTEGER DEFAULT 1,
		is_synced INTEGER DEFAULT 0,
		is_deleted INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		synced_at DATETIME
	);

	-- Таблица очереди синхронизации
	CREATE TABLE IF NOT EXISTS sync_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		secret_id TEXT NOT NULL,
		operation TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (secret_id) REFERENCES secrets(id)
	);

	-- Таблица метаданных синхронизации
	CREATE TABLE IF NOT EXISTS sync_meta (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		last_sync_time DATETIME
	);

	-- Индексы
	CREATE INDEX IF NOT EXISTS idx_secrets_name ON secrets(name);
	CREATE INDEX IF NOT EXISTS idx_secrets_type ON secrets(type);
	CREATE INDEX IF NOT EXISTS idx_secrets_synced ON secrets(is_synced);
	CREATE INDEX IF NOT EXISTS idx_sync_queue_secret ON sync_queue(secret_id);
	`

	_, err := db.Exec(schema)
	return err
}

package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
)

// Session представляет сессию пользователя.
type Session struct {
	UserID        string
	AccessToken   string
	RefreshToken  string
	EncryptionKey []byte
}

// ErrNoSession возвращается когда сессия не найдена.
var ErrNoSession = errors.New("no active session")

// SaveSession сохраняет сессию пользователя.
func (d *DB) SaveSession(session *Session) error {
	query := `
		INSERT OR REPLACE INTO session (id, user_id, access_token, refresh_token, encryption_key)
		VALUES (1, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query, session.UserID, session.AccessToken, session.RefreshToken, session.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// GetSession возвращает текущую сессию.
func (d *DB) GetSession() (*Session, error) {
	query := `
		SELECT user_id, access_token, refresh_token, encryption_key
		FROM session
		WHERE id = 1
	`

	session := &Session{}
	err := d.db.QueryRow(query).Scan(
		&session.UserID,
		&session.AccessToken,
		&session.RefreshToken,
		&session.EncryptionKey,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// UpdateTokens обновляет токены в сессии.
func (d *DB) UpdateTokens(accessToken, refreshToken string) error {
	query := `
		UPDATE session
		SET access_token = ?, refresh_token = ?
		WHERE id = 1
	`

	result, err := d.db.Exec(query, accessToken, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to update tokens: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNoSession
	}

	return nil
}

// DeleteSession удаляет сессию (logout).
func (d *DB) DeleteSession() error {
	_, err := d.db.Exec("DELETE FROM session WHERE id = 1")
	return err
}

// HasSession проверяет наличие активной сессии.
func (d *DB) HasSession() bool {
	var count int
	d.db.QueryRow("SELECT COUNT(*) FROM session WHERE id = 1").Scan(&count)
	return count > 0
}

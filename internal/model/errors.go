package model

import "errors"

// Ошибки аутентификации.
var (
	// ErrUserNotFound возвращается, когда пользователь не найден.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists возвращается при попытке создать пользователя с существующим логином.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidCredentials возвращается при неверном логине или пароле.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidToken возвращается при невалидном или истёкшем токене.
	ErrInvalidToken = errors.New("invalid or expired token")

	// ErrTokenExpired возвращается, когда токен истёк.
	ErrTokenExpired = errors.New("token expired")
)

// Ошибки работы с секретами.
var (
	// ErrSecretNotFound возвращается, когда секрет не найден.
	ErrSecretNotFound = errors.New("secret not found")

	// ErrSecretAlreadyExists возвращается при попытке создать секрет с существующим именем.
	ErrSecretAlreadyExists = errors.New("secret with this name already exists")

	// ErrVersionConflict возвращается при конфликте версий (оптимистичная блокировка).
	ErrVersionConflict = errors.New("version conflict: secret was modified")

	// ErrAccessDenied возвращается при попытке доступа к чужому секрету.
	ErrAccessDenied = errors.New("access denied")
)

// Ошибки синхронизации.
var (
	// ErrSyncConflict возвращается при конфликте данных во время синхронизации.
	ErrSyncConflict = errors.New("sync conflict")
)

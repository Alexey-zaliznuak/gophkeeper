// Package crypto обеспечивает шифрование данных на стороне клиента (E2E encryption).
//
// # Схема шифрования GophKeeper
//
// GophKeeper использует End-to-End шифрование: данные шифруются на клиенте
// ПЕРЕД отправкой на сервер. Сервер хранит только зашифрованные данные и
// НЕ МОЖЕТ их расшифровать без мастер-ключа пользователя.
//
// ## Иерархия ключей
//
//	┌─────────────────────────────────────────────────────────────┐
//	│                    Master Password                          │
//	│              (вводится пользователем)                       │
//	└─────────────────────┬───────────────────────────────────────┘
//	                      │ PBKDF2
//	                      ▼
//	┌─────────────────────────────────────────────────────────────┐
//	│                  Master Key (32 bytes)                      │
//	│           (производный ключ из пароля)                      │
//	└─────────────────────┬───────────────────────────────────────┘
//	                      │ AES-256-GCM
//	                      ▼
//	┌─────────────────────────────────────────────────────────────┐
//	│                  Data Key (32 bytes)                        │
//	│      (случайный ключ, зашифрован Master Key)                │
//	│        Хранится локально в зашифрованном виде               │
//	└─────────────────────┬───────────────────────────────────────┘
//	                      │ AES-256-GCM
//	                      ▼
//	┌─────────────────────────────────────────────────────────────┐
//	│                   Secret Data                               │
//	│       (данные секретов, зашифрованы Data Key)               │
//	└─────────────────────────────────────────────────────────────┘
//
// ## Алгоритмы
//
//   - Деривация ключа: PBKDF2-SHA256 (100,000 итераций)
//   - Шифрование: AES-256-GCM (Galois/Counter Mode)
//   - Хеширование паролей (на сервере): bcrypt
//
// ## Формат зашифрованных данных
//
//	┌──────────┬─────────────────┬─────────────────┐
//	│  Salt    │     Nonce       │   Ciphertext    │
//	│ 32 bytes │    12 bytes     │   variable      │
//	└──────────┴─────────────────┴─────────────────┘
//
// ## Процесс шифрования
//
//  1. Пользователь вводит мастер-пароль при первом входе
//  2. Из пароля генерируется Master Key через PBKDF2
//  3. Генерируется случайный Data Key
//  4. Data Key шифруется Master Key и сохраняется локально
//  5. Все секреты шифруются Data Key перед отправкой на сервер
//
// ## Безопасность
//
//   - Сервер НИКОГДА не видит данные в открытом виде
//   - Мастер-пароль НИКОГДА не покидает клиент
//   - Компрометация сервера НЕ раскрывает данные пользователей
//   - Каждый секрет использует уникальный nonce
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"

	"gophkeeper/internal/model"
)

const (
	// saltSize размер соли для PBKDF2.
	saltSize = 32

	// nonceSize размер nonce для AES-GCM.
	nonceSize = 12

	// keySize размер ключа AES-256.
	keySize = 32

	// pbkdf2Iterations количество итераций PBKDF2.
	pbkdf2Iterations = 100000
)

// Encryptor обеспечивает шифрование и дешифрование данных.
type Encryptor struct {
	dataKey []byte
}

// NewEncryptor создаёт новый экземпляр шифратора с указанным ключом данных.
func NewEncryptor(dataKey []byte) *Encryptor {
	return &Encryptor{dataKey: dataKey}
}

// DeriveKeyFromPassword создаёт ключ шифрования из пароля используя PBKDF2.
// Возвращает ключ и соль (соль нужна для повторной деривации).
func DeriveKeyFromPassword(password string, salt []byte) ([]byte, []byte, error) {
	// Генерируем соль если не передана
	if salt == nil {
		salt = make([]byte, saltSize)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	// Деривация ключа через PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keySize, sha256.New)

	return key, salt, nil
}

// GenerateDataKey генерирует случайный ключ для шифрования данных.
func GenerateDataKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}
	return key, nil
}

// EncryptDataKey шифрует Data Key с помощью Master Key.
// Возвращает зашифрованный ключ в формате: salt + nonce + ciphertext.
func EncryptDataKey(dataKey, masterKey []byte) ([]byte, error) {
	return encrypt(dataKey, masterKey)
}

// DecryptDataKey расшифровывает Data Key с помощью Master Key.
func DecryptDataKey(encryptedKey, masterKey []byte) ([]byte, error) {
	return decrypt(encryptedKey, masterKey)
}

// Encrypt шифрует данные.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	return encrypt(plaintext, e.dataKey)
}

// Decrypt расшифровывает данные.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	return decrypt(ciphertext, e.dataKey)
}

// EncryptCredentials шифрует данные логина/пароля.
func (e *Encryptor) EncryptCredentials(creds *model.CredentialsData) ([]byte, error) {
	data, err := json.Marshal(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}
	return e.Encrypt(data)
}

// DecryptCredentials расшифровывает данные логина/пароля.
func (e *Encryptor) DecryptCredentials(ciphertext []byte) (*model.CredentialsData, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	var creds model.CredentialsData
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// EncryptText шифрует текстовые данные.
func (e *Encryptor) EncryptText(text *model.TextData) ([]byte, error) {
	data, err := json.Marshal(text)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal text: %w", err)
	}
	return e.Encrypt(data)
}

// DecryptText расшифровывает текстовые данные.
func (e *Encryptor) DecryptText(ciphertext []byte) (*model.TextData, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	var text model.TextData
	if err := json.Unmarshal(plaintext, &text); err != nil {
		return nil, fmt.Errorf("failed to unmarshal text: %w", err)
	}

	return &text, nil
}

// EncryptBinary шифрует бинарные данные.
func (e *Encryptor) EncryptBinary(binary *model.BinaryData) ([]byte, error) {
	data, err := json.Marshal(binary)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal binary: %w", err)
	}
	return e.Encrypt(data)
}

// DecryptBinary расшифровывает бинарные данные.
func (e *Encryptor) DecryptBinary(ciphertext []byte) (*model.BinaryData, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	var binary model.BinaryData
	if err := json.Unmarshal(plaintext, &binary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal binary: %w", err)
	}

	return &binary, nil
}

// EncryptCard шифрует данные карты.
func (e *Encryptor) EncryptCard(card *model.CardData) ([]byte, error) {
	data, err := json.Marshal(card)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal card: %w", err)
	}
	return e.Encrypt(data)
}

// DecryptCard расшифровывает данные карты.
func (e *Encryptor) DecryptCard(ciphertext []byte) (*model.CardData, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	var card model.CardData
	if err := json.Unmarshal(plaintext, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal card: %w", err)
	}

	return &card, nil
}

// encrypt шифрует данные с помощью AES-256-GCM.
// Формат результата: nonce (12 bytes) + ciphertext.
func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Результат: nonce + ciphertext (с auth tag)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt расшифровывает данные с помощью AES-256-GCM.
func decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

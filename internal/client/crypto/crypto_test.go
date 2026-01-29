package crypto

import (
	"bytes"
	"testing"

	"gophkeeper/internal/model"
)

func TestDeriveKeyFromPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"simple password", "password123"},
		{"empty password", ""},
		{"unicode password", "пароль🔐"},
		{"long password", "this is a very long password used for key derivation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, salt, err := DeriveKeyFromPassword(tt.password, nil)
			if err != nil {
				t.Fatalf("DeriveKeyFromPassword() error = %v", err)
			}

			if len(key) != 32 {
				t.Errorf("key length = %d, want 32", len(key))
			}

			if len(salt) != 16 {
				t.Errorf("salt length = %d, want 16", len(salt))
			}

			// Проверяем воспроизводимость с той же солью
			key2, _, err := DeriveKeyFromPassword(tt.password, salt)
			if err != nil {
				t.Fatalf("DeriveKeyFromPassword() with salt error = %v", err)
			}

			if !bytes.Equal(key, key2) {
				t.Error("DeriveKeyFromPassword() not reproducible with same salt")
			}
		})
	}
}

func TestDeriveKeyFromPassword_DifferentSalts(t *testing.T) {
	password := "test-password"

	key1, salt1, _ := DeriveKeyFromPassword(password, nil)
	key2, salt2, _ := DeriveKeyFromPassword(password, nil)

	// Разные соли должны давать разные ключи
	if bytes.Equal(salt1, salt2) {
		t.Error("Salts should be different")
	}

	if bytes.Equal(key1, key2) {
		t.Error("Keys should be different with different salts")
	}
}

func TestGenerateDataKey(t *testing.T) {
	key1, err := GenerateDataKey()
	if err != nil {
		t.Fatalf("GenerateDataKey() error = %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("key length = %d, want 32", len(key1))
	}

	// Проверяем уникальность
	key2, _ := GenerateDataKey()
	if bytes.Equal(key1, key2) {
		t.Error("GenerateDataKey() should generate unique keys")
	}
}

func TestEncryptDecryptDataKey(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	masterKey, _, _ := DeriveKeyFromPassword("master-password", nil)

	// Шифруем
	encrypted, err := EncryptDataKey(dataKey, masterKey)
	if err != nil {
		t.Fatalf("EncryptDataKey() error = %v", err)
	}

	// Проверяем что зашифровано
	if bytes.Equal(encrypted, dataKey) {
		t.Error("EncryptDataKey() returned unencrypted key")
	}

	// Расшифровываем
	decrypted, err := DecryptDataKey(encrypted, masterKey)
	if err != nil {
		t.Fatalf("DecryptDataKey() error = %v", err)
	}

	if !bytes.Equal(decrypted, dataKey) {
		t.Error("DecryptDataKey() returned wrong key")
	}
}

func TestEncryptDecryptDataKey_WrongMasterKey(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	masterKey1, _, _ := DeriveKeyFromPassword("password1", nil)
	masterKey2, _, _ := DeriveKeyFromPassword("password2", nil)

	encrypted, _ := EncryptDataKey(dataKey, masterKey1)

	// Пытаемся расшифровать другим ключом
	_, err := DecryptDataKey(encrypted, masterKey2)
	if err == nil {
		t.Error("DecryptDataKey() should fail with wrong master key")
	}
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	encryptor := NewEncryptor(dataKey)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"simple", []byte("Hello, World!")},
		{"empty", []byte("")},
		{"binary", []byte{0x00, 0xFF, 0x01, 0xFE}},
		{"large", bytes.Repeat([]byte("x"), 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Error("Decrypt() returned wrong data")
			}
		})
	}
}

func TestEncryptor_EncryptDecryptCredentials(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	encryptor := NewEncryptor(dataKey)

	creds := &model.CredentialsData{
		Login:    "user@example.com",
		Password: "super-secret-password",
		URL:      "https://example.com",
	}

	encrypted, err := encryptor.EncryptCredentials(creds)
	if err != nil {
		t.Fatalf("EncryptCredentials() error = %v", err)
	}

	decrypted, err := encryptor.DecryptCredentials(encrypted)
	if err != nil {
		t.Fatalf("DecryptCredentials() error = %v", err)
	}

	if decrypted.Login != creds.Login {
		t.Errorf("Login = %v, want %v", decrypted.Login, creds.Login)
	}
	if decrypted.Password != creds.Password {
		t.Errorf("Password = %v, want %v", decrypted.Password, creds.Password)
	}
	if decrypted.URL != creds.URL {
		t.Errorf("URL = %v, want %v", decrypted.URL, creds.URL)
	}
}

func TestEncryptor_EncryptDecryptText(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	encryptor := NewEncryptor(dataKey)

	text := &model.TextData{
		Content: "This is a secret note with\nmultiple lines\nand unicode: 日本語",
	}

	encrypted, err := encryptor.EncryptText(text)
	if err != nil {
		t.Fatalf("EncryptText() error = %v", err)
	}

	decrypted, err := encryptor.DecryptText(encrypted)
	if err != nil {
		t.Fatalf("DecryptText() error = %v", err)
	}

	if decrypted.Content != text.Content {
		t.Errorf("Content = %v, want %v", decrypted.Content, text.Content)
	}
}

func TestEncryptor_EncryptDecryptBinary(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	encryptor := NewEncryptor(dataKey)

	binary := &model.BinaryData{
		FileName: "test.bin",
		MimeType: "application/octet-stream",
		Data:     []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
	}

	encrypted, err := encryptor.EncryptBinary(binary)
	if err != nil {
		t.Fatalf("EncryptBinary() error = %v", err)
	}

	decrypted, err := encryptor.DecryptBinary(encrypted)
	if err != nil {
		t.Fatalf("DecryptBinary() error = %v", err)
	}

	if decrypted.FileName != binary.FileName {
		t.Errorf("FileName = %v, want %v", decrypted.FileName, binary.FileName)
	}
	if !bytes.Equal(decrypted.Data, binary.Data) {
		t.Error("Data mismatch")
	}
}

func TestEncryptor_EncryptDecryptCard(t *testing.T) {
	dataKey, _ := GenerateDataKey()
	encryptor := NewEncryptor(dataKey)

	card := &model.CardData{
		Number:      "4111111111111111",
		HolderName:  "JOHN DOE",
		ExpiryMonth: 12,
		ExpiryYear:  2025,
		CVV:         "123",
	}

	encrypted, err := encryptor.EncryptCard(card)
	if err != nil {
		t.Fatalf("EncryptCard() error = %v", err)
	}

	decrypted, err := encryptor.DecryptCard(encrypted)
	if err != nil {
		t.Fatalf("DecryptCard() error = %v", err)
	}

	if decrypted.Number != card.Number {
		t.Errorf("Number = %v, want %v", decrypted.Number, card.Number)
	}
	if decrypted.HolderName != card.HolderName {
		t.Errorf("HolderName = %v, want %v", decrypted.HolderName, card.HolderName)
	}
	if decrypted.ExpiryMonth != card.ExpiryMonth {
		t.Errorf("ExpiryMonth = %v, want %v", decrypted.ExpiryMonth, card.ExpiryMonth)
	}
	if decrypted.ExpiryYear != card.ExpiryYear {
		t.Errorf("ExpiryYear = %v, want %v", decrypted.ExpiryYear, card.ExpiryYear)
	}
	if decrypted.CVV != card.CVV {
		t.Errorf("CVV = %v, want %v", decrypted.CVV, card.CVV)
	}
}

func TestEncryptor_DecryptWithWrongKey(t *testing.T) {
	key1, _ := GenerateDataKey()
	key2, _ := GenerateDataKey()

	encryptor1 := NewEncryptor(key1)
	encryptor2 := NewEncryptor(key2)

	creds := &model.CredentialsData{Login: "user", Password: "pass"}
	encrypted, _ := encryptor1.EncryptCredentials(creds)

	// Пытаемся расшифровать другим ключом
	_, err := encryptor2.DecryptCredentials(encrypted)
	if err == nil {
		t.Error("DecryptCredentials() should fail with wrong key")
	}
}

func TestNewEncryptor(t *testing.T) {
	key, _ := GenerateDataKey()
	encryptor := NewEncryptor(key)

	if encryptor == nil {
		t.Error("NewEncryptor() returned nil")
	}
}

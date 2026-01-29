package crypto

import (
	"bytes"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"simple password", "password123"},
		{"empty password", ""},
		{"long password", "this is a very long password that should still work correctly"},
		{"special chars", "p@$$w0rd!#$%^&*()"},
		{"unicode", "пароль密码🔐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}

			if hash == "" {
				t.Error("HashPassword() returned empty hash")
			}

			if hash == tt.password {
				t.Error("HashPassword() returned unhashed password")
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testPassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "wrongPassword", hash, false},
		{"empty password", "", hash, false},
		{"invalid hash", password, "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPassword(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"simple token", "abc123"},
		{"empty token", ""},
		{"long token", "verylongtokenthatshouldbehashed1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashToken(tt.token)

			if hash == "" {
				t.Error("HashToken() returned empty hash")
			}

			// Проверяем детерминированность
			hash2 := HashToken(tt.token)
			if hash != hash2 {
				t.Error("HashToken() is not deterministic")
			}

			// Проверяем что разные токены дают разные хеши
			if tt.token != "" {
				differentHash := HashToken(tt.token + "x")
				if hash == differentHash {
					t.Error("HashToken() collision detected")
				}
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"simple text", []byte("Hello, World!")},
		{"empty", []byte("")},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}},
		{"large data", bytes.Repeat([]byte("x"), 10000)},
		{"unicode", []byte("Привет мир! 你好世界! 🔐")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Проверяем что шифротекст отличается от открытого текста
			if len(tt.plaintext) > 0 && bytes.Equal(ciphertext, tt.plaintext) {
				t.Error("Encrypt() returned plaintext")
			}

			// Расшифровываем
			decrypted, err := Decrypt(ciphertext, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptDecrypt_DifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 1)
	}

	plaintext := []byte("secret data")

	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Пытаемся расшифровать другим ключом
	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"too short", 16},
		{"too long", 64},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := Encrypt([]byte("test"), key)
			if err == nil {
				t.Error("Encrypt() should fail with invalid key size")
			}
		})
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{"too short", []byte{1, 2, 3}},
		{"empty", []byte{}},
		{"corrupted", append(make([]byte, 12), []byte("corrupted")...)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext, key)
			if err == nil {
				t.Error("Decrypt() should fail with invalid ciphertext")
			}
		})
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
		{"1 byte", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes1, err := GenerateRandomBytes(tt.n)
			if err != nil {
				t.Fatalf("GenerateRandomBytes() error = %v", err)
			}

			if len(bytes1) != tt.n {
				t.Errorf("GenerateRandomBytes() length = %d, want %d", len(bytes1), tt.n)
			}

			// Проверяем что генерируются разные значения
			bytes2, _ := GenerateRandomBytes(tt.n)
			if bytes.Equal(bytes1, bytes2) {
				t.Error("GenerateRandomBytes() returned same values")
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str1, err := GenerateRandomString(tt.n)
			if err != nil {
				t.Fatalf("GenerateRandomString() error = %v", err)
			}

			// Hex encoding doubles the length
			if len(str1) != tt.n*2 {
				t.Errorf("GenerateRandomString() length = %d, want %d", len(str1), tt.n*2)
			}

			// Проверяем уникальность
			str2, _ := GenerateRandomString(tt.n)
			if str1 == str2 {
				t.Error("GenerateRandomString() returned same values")
			}
		})
	}
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("same plaintext")

	// Шифруем один и тот же текст несколько раз
	ciphertexts := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		ct, err := Encrypt(plaintext, key)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		ciphertexts[i] = ct
	}

	// Проверяем что все шифротексты уникальны (из-за разных nonce)
	for i := 0; i < len(ciphertexts); i++ {
		for j := i + 1; j < len(ciphertexts); j++ {
			if bytes.Equal(ciphertexts[i], ciphertexts[j]) {
				t.Error("Encrypt() produced identical ciphertexts (nonce reuse)")
			}
		}
	}
}

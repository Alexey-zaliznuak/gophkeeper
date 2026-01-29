package model

import (
	"encoding/json"
	"testing"
)

func TestCredentialsData_JSON(t *testing.T) {
	creds := &CredentialsData{
		Login:    "user@example.com",
		Password: "secret123",
		URL:      "https://example.com",
	}

	// Сериализация
	data, err := json.Marshal(creds)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Десериализация
	var decoded CredentialsData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Login != creds.Login {
		t.Errorf("Login = %v, want %v", decoded.Login, creds.Login)
	}
	if decoded.Password != creds.Password {
		t.Errorf("Password = %v, want %v", decoded.Password, creds.Password)
	}
	if decoded.URL != creds.URL {
		t.Errorf("URL = %v, want %v", decoded.URL, creds.URL)
	}
}

func TestCredentialsData_JSON_OmitEmpty(t *testing.T) {
	creds := &CredentialsData{
		Login:    "user",
		Password: "pass",
		// URL не задан
	}

	data, _ := json.Marshal(creds)
	jsonStr := string(data)

	// URL должен быть опущен
	if contains(jsonStr, "url") {
		t.Error("Empty URL should be omitted from JSON")
	}
}

func TestTextData_JSON(t *testing.T) {
	text := &TextData{
		Content: "This is a secret note\nwith multiple lines",
	}

	data, err := json.Marshal(text)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded TextData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Content != text.Content {
		t.Errorf("Content = %v, want %v", decoded.Content, text.Content)
	}
}

func TestBinaryData_JSON(t *testing.T) {
	binary := &BinaryData{
		FileName: "document.pdf",
		MimeType: "application/pdf",
		Data:     []byte{0x25, 0x50, 0x44, 0x46}, // PDF magic bytes
	}

	data, err := json.Marshal(binary)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded BinaryData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.FileName != binary.FileName {
		t.Errorf("FileName = %v, want %v", decoded.FileName, binary.FileName)
	}
	if decoded.MimeType != binary.MimeType {
		t.Errorf("MimeType = %v, want %v", decoded.MimeType, binary.MimeType)
	}
	if len(decoded.Data) != len(binary.Data) {
		t.Errorf("Data length = %v, want %v", len(decoded.Data), len(binary.Data))
	}
}

func TestCardData_JSON(t *testing.T) {
	card := &CardData{
		Number:      "4111111111111111",
		HolderName:  "JOHN DOE",
		ExpiryMonth: 12,
		ExpiryYear:  2025,
		CVV:         "123",
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded CardData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Number != card.Number {
		t.Errorf("Number = %v, want %v", decoded.Number, card.Number)
	}
	if decoded.HolderName != card.HolderName {
		t.Errorf("HolderName = %v, want %v", decoded.HolderName, card.HolderName)
	}
	if decoded.ExpiryMonth != card.ExpiryMonth {
		t.Errorf("ExpiryMonth = %v, want %v", decoded.ExpiryMonth, card.ExpiryMonth)
	}
	if decoded.ExpiryYear != card.ExpiryYear {
		t.Errorf("ExpiryYear = %v, want %v", decoded.ExpiryYear, card.ExpiryYear)
	}
	if decoded.CVV != card.CVV {
		t.Errorf("CVV = %v, want %v", decoded.CVV, card.CVV)
	}
}

func TestCardData_JSONFieldNames(t *testing.T) {
	card := &CardData{
		Number:      "4111111111111111",
		HolderName:  "JOHN DOE",
		ExpiryMonth: 12,
		ExpiryYear:  2025,
		CVV:         "123",
	}

	data, _ := json.Marshal(card)
	jsonStr := string(data)

	// Проверяем snake_case имена полей
	expectedFields := []string{"number", "holder_name", "expiry_month", "expiry_year", "cvv"}
	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON should contain field %q", field)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

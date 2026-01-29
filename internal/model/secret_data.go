package model

// CredentialsData содержит данные для типа SecretTypeCredentials.
type CredentialsData struct {
	// Login - логин/имя пользователя.
	Login string `json:"login"`

	// Password - пароль.
	Password string `json:"password"`

	// URL - адрес сайта/сервиса (опционально).
	URL string `json:"url,omitempty"`
}

// TextData содержит данные для типа SecretTypeText.
type TextData struct {
	// Content - текстовое содержимое.
	Content string `json:"content"`
}

// BinaryData содержит данные для типа SecretTypeBinary.
type BinaryData struct {
	// FileName - имя файла.
	FileName string `json:"file_name"`

	// MimeType - MIME-тип файла.
	MimeType string `json:"mime_type,omitempty"`

	// Data - бинарные данные.
	Data []byte `json:"data"`
}

// CardData содержит данные банковской карты.
type CardData struct {
	// Number - номер карты.
	Number string `json:"number"`

	// HolderName - имя держателя карты.
	HolderName string `json:"holder_name"`

	// ExpiryMonth - месяц окончания действия (1-12).
	ExpiryMonth int `json:"expiry_month"`

	// ExpiryYear - год окончания действия (например, 2025).
	ExpiryYear int `json:"expiry_year"`

	// CVV - код безопасности.
	CVV string `json:"cvv"`
}

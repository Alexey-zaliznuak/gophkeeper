package service

import (
	"context"
	"fmt"

	"gophkeeper/internal/client/api"
	"gophkeeper/internal/client/crypto"
	"gophkeeper/internal/client/storage/sqlite"
	"gophkeeper/internal/model"
	"gophkeeper/internal/server/grpc/pb"
)

// SecretService сервис работы с секретами на клиенте.
type SecretService struct {
	client    *api.Client
	storage   *sqlite.DB
	encryptor *crypto.Encryptor
}

// NewSecretService создаёт новый сервис секретов.
func NewSecretService(client *api.Client, storage *sqlite.DB, encryptor *crypto.Encryptor) *SecretService {
	return &SecretService{
		client:    client,
		storage:   storage,
		encryptor: encryptor,
	}
}

// ==================== Credentials ====================

// AddCredentials добавляет логин/пароль.
func (s *SecretService) AddCredentials(ctx context.Context, name string, creds *model.CredentialsData, metadata map[string]string) error {
	// Шифруем данные
	encryptedData, err := s.encryptor.EncryptCredentials(creds)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	return s.createSecret(ctx, name, model.SecretTypeCredentials, encryptedData, metadata)
}

// GetCredentials возвращает расшифрованные учётные данные.
func (s *SecretService) GetCredentials(ctx context.Context, name string) (*model.CredentialsData, map[string]string, error) {
	secret, err := s.storage.GetSecretByName(name)
	if err != nil {
		return nil, nil, err
	}

	if secret.Type != model.SecretTypeCredentials {
		return nil, nil, fmt.Errorf("secret is not credentials type")
	}

	creds, err := s.encryptor.DecryptCredentials(secret.EncryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	return creds, secret.Metadata, nil
}

// ==================== Text ====================

// AddText добавляет текстовую заметку.
func (s *SecretService) AddText(ctx context.Context, name string, text *model.TextData, metadata map[string]string) error {
	encryptedData, err := s.encryptor.EncryptText(text)
	if err != nil {
		return fmt.Errorf("failed to encrypt text: %w", err)
	}

	return s.createSecret(ctx, name, model.SecretTypeText, encryptedData, metadata)
}

// GetText возвращает расшифрованную текстовую заметку.
func (s *SecretService) GetText(ctx context.Context, name string) (*model.TextData, map[string]string, error) {
	secret, err := s.storage.GetSecretByName(name)
	if err != nil {
		return nil, nil, err
	}

	if secret.Type != model.SecretTypeText {
		return nil, nil, fmt.Errorf("secret is not text type")
	}

	text, err := s.encryptor.DecryptText(secret.EncryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt text: %w", err)
	}

	return text, secret.Metadata, nil
}

// ==================== Binary ====================

// AddBinary добавляет бинарные данные.
func (s *SecretService) AddBinary(ctx context.Context, name string, binary *model.BinaryData, metadata map[string]string) error {
	encryptedData, err := s.encryptor.EncryptBinary(binary)
	if err != nil {
		return fmt.Errorf("failed to encrypt binary: %w", err)
	}

	return s.createSecret(ctx, name, model.SecretTypeBinary, encryptedData, metadata)
}

// GetBinary возвращает расшифрованные бинарные данные.
func (s *SecretService) GetBinary(ctx context.Context, name string) (*model.BinaryData, map[string]string, error) {
	secret, err := s.storage.GetSecretByName(name)
	if err != nil {
		return nil, nil, err
	}

	if secret.Type != model.SecretTypeBinary {
		return nil, nil, fmt.Errorf("secret is not binary type")
	}

	binary, err := s.encryptor.DecryptBinary(secret.EncryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt binary: %w", err)
	}

	return binary, secret.Metadata, nil
}

// ==================== Card ====================

// AddCard добавляет банковскую карту.
func (s *SecretService) AddCard(ctx context.Context, name string, card *model.CardData, metadata map[string]string) error {
	encryptedData, err := s.encryptor.EncryptCard(card)
	if err != nil {
		return fmt.Errorf("failed to encrypt card: %w", err)
	}

	return s.createSecret(ctx, name, model.SecretTypeCard, encryptedData, metadata)
}

// GetCard возвращает расшифрованные данные карты.
func (s *SecretService) GetCard(ctx context.Context, name string) (*model.CardData, map[string]string, error) {
	secret, err := s.storage.GetSecretByName(name)
	if err != nil {
		return nil, nil, err
	}

	if secret.Type != model.SecretTypeCard {
		return nil, nil, fmt.Errorf("secret is not card type")
	}

	card, err := s.encryptor.DecryptCard(secret.EncryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt card: %w", err)
	}

	return card, secret.Metadata, nil
}

// ==================== Common ====================

// createSecret создаёт секрет локально.
func (s *SecretService) createSecret(ctx context.Context, name string, secretType model.SecretType, encryptedData []byte, metadata map[string]string) error {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	secret := &sqlite.LocalSecret{
		Name:          name,
		Type:          secretType,
		EncryptedData: encryptedData,
		Metadata:      metadata,
		Version:       1,
		IsSynced:      false,
	}

	if err := s.storage.CreateSecret(secret); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// Delete удаляет секрет.
func (s *SecretService) Delete(ctx context.Context, name string) error {
	secret, err := s.storage.GetSecretByName(name)
	if err != nil {
		return err
	}

	return s.storage.DeleteSecret(secret.ID)
}

// List возвращает список всех секретов.
func (s *SecretService) List(ctx context.Context) ([]*sqlite.LocalSecret, error) {
	return s.storage.ListSecrets()
}

// ListByType возвращает секреты по типу.
func (s *SecretService) ListByType(ctx context.Context, secretType model.SecretType) ([]*sqlite.LocalSecret, error) {
	return s.storage.ListSecretsByType(secretType)
}

// Sync синхронизирует секреты с сервером.
func (s *SecretService) Sync(ctx context.Context) error {
	// 1. Получаем изменения с сервера
	lastSync, _ := s.storage.GetLastSyncTime()

	var req pb.GetChangesRequest
	if lastSync != nil {
		// Используем timestamppb напрямую
	}

	changes, err := s.client.GetChanges(ctx, &req)
	if err != nil {
		return fmt.Errorf("failed to get changes: %w", err)
	}

	// 2. Применяем изменения с сервера
	for _, secret := range changes.Secrets {
		localSecret := &sqlite.LocalSecret{
			ID:            secret.Id,
			ServerID:      secret.Id,
			Name:          secret.Name,
			Type:          model.SecretType(secret.Type),
			EncryptedData: secret.EncryptedData,
			Metadata:      secret.Metadata,
			Version:       secret.Version,
			IsSynced:      true,
		}

		if err := s.storage.UpsertFromServer(localSecret); err != nil {
			return fmt.Errorf("failed to upsert secret: %w", err)
		}
	}

	// 3. Удаляем локально удалённые на сервере
	for _, deletedID := range changes.DeletedIds {
		_ = s.storage.DeleteByServerID(deletedID)
	}

	// 4. Отправляем локальные изменения на сервер
	queue, err := s.storage.GetSyncQueue()
	if err != nil {
		return fmt.Errorf("failed to get sync queue: %w", err)
	}

	if len(queue) > 0 {
		var pushChanges []*pb.SecretChange

		for _, item := range queue {
			secret, err := s.storage.GetSecretByID(item.SecretID)
			if err != nil && item.Operation != "delete" {
				continue
			}

			change := &pb.SecretChange{
				ClientId: item.SecretID,
			}

			switch item.Operation {
			case "create":
				change.Operation = pb.ChangeOperation_CHANGE_OPERATION_CREATE
				change.Secret = &pb.Secret{
					Name:          secret.Name,
					Type:          pb.SecretType(secret.Type),
					EncryptedData: secret.EncryptedData,
					Metadata:      secret.Metadata,
				}
			case "update":
				change.Operation = pb.ChangeOperation_CHANGE_OPERATION_UPDATE
				change.Secret = &pb.Secret{
					Id:            secret.ServerID,
					Name:          secret.Name,
					Type:          pb.SecretType(secret.Type),
					EncryptedData: secret.EncryptedData,
					Metadata:      secret.Metadata,
					Version:       secret.Version,
				}
			case "delete":
				change.Operation = pb.ChangeOperation_CHANGE_OPERATION_DELETE
				change.SecretId = item.SecretID
			}

			pushChanges = append(pushChanges, change)
		}

		if len(pushChanges) > 0 {
			resp, err := s.client.PushChanges(ctx, &pb.PushChangesRequest{Changes: pushChanges})
			if err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}

			// Обрабатываем результаты
			for _, result := range resp.Results {
				if result.Success {
					_ = s.storage.MarkAsSynced(result.ClientId, result.ServerId)
				}
			}
		}

		// Очищаем очередь
		_ = s.storage.ClearSyncQueue()
	}

	// 5. Сохраняем время синхронизации
	if changes.ServerTime != nil {
		_ = s.storage.SetLastSyncTime(changes.ServerTime.AsTime())
	}

	return nil
}

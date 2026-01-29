// Package secret реализует сервис работы с секретами.
package secret

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"gophkeeper/internal/model"
	"gophkeeper/internal/server/repository"
)

// Service реализует сервис работы с секретами.
type Service struct {
	secretRepo repository.SecretRepository
	log        *zap.Logger
}

// NewService создаёт новый сервис секретов.
func NewService(secretRepo repository.SecretRepository, log *zap.Logger) *Service {
	return &Service{
		secretRepo: secretRepo,
		log:        log,
	}
}

// Create создаёт новый секрет.
func (s *Service) Create(ctx context.Context, userID string, secret *model.Secret) (*model.Secret, error) {
	secret.UserID = userID

	if err := s.secretRepo.Create(ctx, secret); err != nil {
		return nil, err
	}

	s.log.Debug("secret created",
		zap.String("user_id", userID),
		zap.String("secret_id", secret.ID),
		zap.String("name", secret.Name),
	)

	return secret, nil
}

// Update обновляет существующий секрет с проверкой версии.
func (s *Service) Update(ctx context.Context, userID string, secret *model.Secret, expectedVersion int64) (*model.Secret, error) {
	// Проверяем, что секрет принадлежит пользователю
	existing, err := s.secretRepo.GetByID(ctx, userID, secret.ID)
	if err != nil {
		return nil, err
	}

	// Проверяем владельца
	if existing.UserID != userID {
		return nil, model.ErrAccessDenied
	}

	// Устанавливаем версию для оптимистичной блокировки
	secret.UserID = userID
	secret.Version = expectedVersion

	if err := s.secretRepo.Update(ctx, secret); err != nil {
		return nil, err
	}

	s.log.Debug("secret updated",
		zap.String("user_id", userID),
		zap.String("secret_id", secret.ID),
		zap.Int64("version", secret.Version),
	)

	return secret, nil
}

// Delete удаляет секрет.
func (s *Service) Delete(ctx context.Context, userID, secretID string) error {
	if err := s.secretRepo.Delete(ctx, userID, secretID); err != nil {
		return err
	}

	s.log.Debug("secret deleted",
		zap.String("user_id", userID),
		zap.String("secret_id", secretID),
	)

	return nil
}

// Get возвращает секрет по ID.
func (s *Service) Get(ctx context.Context, userID, secretID string) (*model.Secret, error) {
	secret, err := s.secretRepo.GetByID(ctx, userID, secretID)
	if err != nil {
		return nil, err
	}

	// Дополнительная проверка владельца
	if secret.UserID != userID {
		return nil, model.ErrAccessDenied
	}

	return secret, nil
}

// List возвращает все секреты пользователя.
func (s *Service) List(ctx context.Context, userID string) ([]*model.Secret, error) {
	secrets, err := s.secretRepo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	return secrets, nil
}

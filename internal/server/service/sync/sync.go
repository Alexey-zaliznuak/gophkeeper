// Package sync реализует сервис синхронизации данных.
package sync

import (
	"context"
	"time"

	"go.uber.org/zap"

	"gophkeeper/internal/model"
	"gophkeeper/internal/server/repository"
	"gophkeeper/internal/server/service"
)

// Service реализует сервис синхронизации.
type Service struct {
	secretRepo repository.SecretRepository
	log        *zap.Logger
}

// NewService создаёт новый сервис синхронизации.
func NewService(secretRepo repository.SecretRepository, log *zap.Logger) *Service {
	return &Service{
		secretRepo: secretRepo,
		log:        log,
	}
}

// GetChanges возвращает изменения с указанного времени.
func (s *Service) GetChanges(ctx context.Context, userID string, since *time.Time) (*service.SyncChanges, error) {
	var secrets []*model.Secret
	var deletedIDs []string
	var err error

	if since == nil {
		// Первая синхронизация - получаем все секреты
		secrets, err = s.secretRepo.List(ctx, userID)
		if err != nil {
			return nil, err
		}
	} else {
		// Инкрементальная синхронизация
		secrets, err = s.secretRepo.ListChangedSince(ctx, userID, *since)
		if err != nil {
			return nil, err
		}

		deletedIDs, err = s.secretRepo.ListDeletedSince(ctx, userID, *since)
		if err != nil {
			return nil, err
		}
	}

	s.log.Debug("sync get changes",
		zap.String("user_id", userID),
		zap.Int("secrets_count", len(secrets)),
		zap.Int("deleted_count", len(deletedIDs)),
	)

	return &service.SyncChanges{
		Secrets:    secrets,
		DeletedIDs: deletedIDs,
		ServerTime: time.Now(),
	}, nil
}

// PushChanges применяет изменения от клиента.
func (s *Service) PushChanges(ctx context.Context, userID string, changes []*service.SecretChange) ([]*service.ChangeResult, error) {
	results := make([]*service.ChangeResult, 0, len(changes))

	for _, change := range changes {
		result := &service.ChangeResult{
			ClientID: change.ClientID,
			Success:  true,
		}

		switch change.Operation {
		case service.ChangeOperationCreate:
			change.Secret.UserID = userID
			if err := s.secretRepo.Create(ctx, change.Secret); err != nil {
				result.Success = false
				result.ErrorMessage = err.Error()
			} else {
				result.ServerID = change.Secret.ID
			}

		case service.ChangeOperationUpdate:
			change.Secret.UserID = userID
			if err := s.secretRepo.Update(ctx, change.Secret); err != nil {
				result.Success = false
				result.ErrorMessage = err.Error()

				// При конфликте версий возвращаем серверную версию
				if err == model.ErrVersionConflict {
					serverSecret, _ := s.secretRepo.GetByID(ctx, userID, change.Secret.ID)
					result.ConflictSecret = serverSecret
				}
			} else {
				result.ServerID = change.Secret.ID
			}

		case service.ChangeOperationDelete:
			if err := s.secretRepo.Delete(ctx, userID, change.SecretID); err != nil {
				result.Success = false
				result.ErrorMessage = err.Error()
			}
			result.ServerID = change.SecretID
		}

		results = append(results, result)
	}

	s.log.Debug("sync push changes",
		zap.String("user_id", userID),
		zap.Int("changes_count", len(changes)),
	)

	return results, nil
}

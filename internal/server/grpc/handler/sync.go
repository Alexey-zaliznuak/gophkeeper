package handler

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"gophkeeper/internal/model"
	"gophkeeper/internal/server/grpc/interceptor"
	"gophkeeper/internal/server/grpc/pb"
	"gophkeeper/internal/server/service"
)

// SyncHandler обработчик SyncService.
type SyncHandler struct {
	pb.UnimplementedSyncServiceServer
	syncService service.SyncService
}

// NewSyncHandler создаёт новый обработчик синхронизации.
func NewSyncHandler(syncService service.SyncService) *SyncHandler {
	return &SyncHandler{syncService: syncService}
}

// GetChanges возвращает изменения секретов с указанного времени.
func (h *SyncHandler) GetChanges(ctx context.Context, req *pb.GetChangesRequest) (*pb.GetChangesResponse, error) {
	userID, err := interceptor.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var since *timestamppb.Timestamp
	if req.Since != nil {
		since = req.Since
	}

	var sinceTime *time.Time
	if since != nil {
		t := since.AsTime()
		sinceTime = &t
	}

	changes, err := h.syncService.GetChanges(ctx, userID, sinceTime)
	if err != nil {
		return nil, mapSecretError(err)
	}

	protoSecrets := make([]*pb.Secret, len(changes.Secrets))
	for i, s := range changes.Secrets {
		protoSecrets[i] = secretToProto(s)
	}

	return &pb.GetChangesResponse{
		Secrets:    protoSecrets,
		DeletedIds: changes.DeletedIDs,
		ServerTime: timestamppb.New(changes.ServerTime),
	}, nil
}

// PushChanges отправляет локальные изменения на сервер.
func (h *SyncHandler) PushChanges(ctx context.Context, req *pb.PushChangesRequest) (*pb.PushChangesResponse, error) {
	userID, err := interceptor.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	changes := make([]*service.SecretChange, len(req.GetChanges()))
	for i, c := range req.GetChanges() {
		change := &service.SecretChange{
			Operation: protoOperationToService(c.GetOperation()),
			ClientID:  c.GetClientId(),
			SecretID:  c.GetSecretId(),
		}

		if c.GetSecret() != nil {
			change.Secret = protoToSecret(c.GetSecret())
		}

		changes[i] = change
	}

	results, err := h.syncService.PushChanges(ctx, userID, changes)
	if err != nil {
		return nil, mapSecretError(err)
	}

	protoResults := make([]*pb.ChangeResult, len(results))
	for i, r := range results {
		protoResult := &pb.ChangeResult{
			ClientId:     r.ClientID,
			ServerId:     r.ServerID,
			Success:      r.Success,
			ErrorMessage: r.ErrorMessage,
		}

		if r.ConflictSecret != nil {
			protoResult.ConflictSecret = secretToProto(r.ConflictSecret)
		}

		protoResults[i] = protoResult
	}

	return &pb.PushChangesResponse{
		Results:    protoResults,
		ServerTime: timestamppb.Now(),
	}, nil
}

// protoToSecret преобразует proto секрет в модель.
func protoToSecret(p *pb.Secret) *model.Secret {
	return &model.Secret{
		ID:            p.GetId(),
		Name:          p.GetName(),
		Type:          model.SecretType(p.GetType()),
		EncryptedData: p.GetEncryptedData(),
		Metadata:      p.GetMetadata(),
		Version:       p.GetVersion(),
	}
}

// protoOperationToService преобразует proto операцию в сервисную.
func protoOperationToService(op pb.ChangeOperation) service.ChangeOperation {
	switch op {
	case pb.ChangeOperation_CHANGE_OPERATION_CREATE:
		return service.ChangeOperationCreate
	case pb.ChangeOperation_CHANGE_OPERATION_UPDATE:
		return service.ChangeOperationUpdate
	case pb.ChangeOperation_CHANGE_OPERATION_DELETE:
		return service.ChangeOperationDelete
	default:
		return service.ChangeOperationCreate
	}
}

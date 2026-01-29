package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gophkeeper/internal/model"
	"gophkeeper/internal/server/grpc/interceptor"
	"gophkeeper/internal/server/grpc/pb"
	"gophkeeper/internal/server/service"
)

// SecretHandler обработчик SecretService.
type SecretHandler struct {
	pb.UnimplementedSecretServiceServer
	secretService service.SecretService
}

// NewSecretHandler создаёт новый обработчик секретов.
func NewSecretHandler(secretService service.SecretService) *SecretHandler {
	return &SecretHandler{secretService: secretService}
}

// Create создаёт новый секрет.
func (h *SecretHandler) Create(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	userID, err := interceptor.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	secret := &model.Secret{
		Name:          req.GetName(),
		Type:          model.SecretType(req.GetType()),
		EncryptedData: req.GetEncryptedData(),
		Metadata:      req.GetMetadata(),
	}

	created, err := h.secretService.Create(ctx, userID, secret)
	if err != nil {
		return nil, mapSecretError(err)
	}

	return &pb.CreateSecretResponse{
		Secret: secretToProto(created),
	}, nil
}

// Get возвращает секрет по ID.
func (h *SecretHandler) Get(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {
	userID, err := interceptor.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	secret, err := h.secretService.Get(ctx, userID, req.GetId())
	if err != nil {
		return nil, mapSecretError(err)
	}

	return &pb.GetSecretResponse{
		Secret: secretToProto(secret),
	}, nil
}

// Update обновляет существующий секрет.
func (h *SecretHandler) Update(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	userID, err := interceptor.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Получаем текущий секрет для обновления
	existing, err := h.secretService.Get(ctx, userID, req.GetId())
	if err != nil {
		return nil, mapSecretError(err)
	}

	// Обновляем только переданные поля
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.EncryptedData != nil {
		existing.EncryptedData = req.EncryptedData
	}
	if len(req.GetMetadata()) > 0 {
		existing.Metadata = req.GetMetadata()
	}

	updated, err := h.secretService.Update(ctx, userID, existing, req.GetExpectedVersion())
	if err != nil {
		return nil, mapSecretError(err)
	}

	return &pb.UpdateSecretResponse{
		Secret: secretToProto(updated),
	}, nil
}

// Delete удаляет секрет.
func (h *SecretHandler) Delete(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	userID, err := interceptor.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := h.secretService.Delete(ctx, userID, req.GetId()); err != nil {
		return nil, mapSecretError(err)
	}

	return &pb.DeleteSecretResponse{}, nil
}

// List возвращает список всех секретов пользователя.
func (h *SecretHandler) List(ctx context.Context, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	userID, err := interceptor.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	secrets, err := h.secretService.List(ctx, userID)
	if err != nil {
		return nil, mapSecretError(err)
	}

	protoSecrets := make([]*pb.Secret, len(secrets))
	for i, s := range secrets {
		protoSecrets[i] = secretToProto(s)
	}

	return &pb.ListSecretsResponse{
		Secrets: protoSecrets,
	}, nil
}

// secretToProto преобразует модель секрета в proto.
func secretToProto(s *model.Secret) *pb.Secret {
	return &pb.Secret{
		Id:            s.ID,
		Name:          s.Name,
		Type:          pb.SecretType(s.Type),
		EncryptedData: s.EncryptedData,
		Metadata:      s.Metadata,
		Version:       s.Version,
		CreatedAt:     timestamppb.New(s.CreatedAt),
		UpdatedAt:     timestamppb.New(s.UpdatedAt),
	}
}

// mapSecretError преобразует ошибки секретов в gRPC статусы.
func mapSecretError(err error) error {
	switch err {
	case model.ErrSecretNotFound:
		return status.Error(codes.NotFound, "secret not found")
	case model.ErrSecretAlreadyExists:
		return status.Error(codes.AlreadyExists, "secret with this name already exists")
	case model.ErrVersionConflict:
		return status.Error(codes.Aborted, "version conflict: secret was modified")
	case model.ErrAccessDenied:
		return status.Error(codes.PermissionDenied, "access denied")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

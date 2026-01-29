package interceptor

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor interceptor для логирования запросов.
type LoggingInterceptor struct {
	log *zap.Logger
}

// NewLoggingInterceptor создаёт новый interceptor логирования.
func NewLoggingInterceptor(log *zap.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{log: log}
}

// Unary возвращает unary interceptor для логирования.
func (i *LoggingInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		// Определяем уровень логирования
		logFunc := i.log.Info
		if code != codes.OK {
			logFunc = i.log.Warn
		}

		logFunc("gRPC request",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", code.String()),
		)

		return resp, err
	}
}

// Stream возвращает stream interceptor для логирования.
func (i *LoggingInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
		}

		logFunc := i.log.Info
		if code != codes.OK {
			logFunc = i.log.Warn
		}

		logFunc("gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", code.String()),
		)

		return err
	}
}

package grpcx

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewServer returns a gRPC server with recovery and logging interceptors.
func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	chain := grpc.ChainUnaryInterceptor(recoveryInterceptor(), loggingInterceptor())
	return grpc.NewServer(append([]grpc.ServerOption{chain}, opts...)...)
}

func recoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("grpc panic", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			slog.Error("grpc error", "method", info.FullMethod, "error", err)
		} else {
			slog.Debug("grpc ok", "method", info.FullMethod)
		}
		return resp, err
	}
}

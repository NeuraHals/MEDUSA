package grpc

import (
	"context"

	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

// chainInterceptors chains multiple unary server interceptors into one.
func chainInterceptors(interceptors ...googlegrpc.UnaryServerInterceptor) googlegrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *googlegrpc.UnaryServerInfo, handler googlegrpc.UnaryHandler) (interface{}, error) {
		h := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			i := i
			prev := h
			h = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptors[i](ctx, req, info, prev)
			}
		}
		return h(ctx, req)
	}
}

func loggingInterceptor(log *zap.Logger) googlegrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *googlegrpc.UnaryServerInfo, handler googlegrpc.UnaryHandler) (interface{}, error) {
		log.Info("gRPC inbound", zap.String("method", info.FullMethod))
		resp, err := handler(ctx, req)
		if err != nil {
			log.Error("gRPC error", zap.String("method", info.FullMethod), zap.Error(err))
		}
		return resp, err
	}
}

func spiffeInterceptor(log *zap.Logger) googlegrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *googlegrpc.UnaryServerInfo, handler googlegrpc.UnaryHandler) (interface{}, error) {
		// In production: extract SPIFFE SVID from peer certificate in ctx
		// and verify caller is spiffe://medusa/orchestrator-agent
		// Enforced by Envoy mTLS sidecar — this is a secondary defense-in-depth check.
		log.Debug("SPIFFE identity check (stub)", zap.String("method", info.FullMethod))
		return handler(ctx, req)
	}
}

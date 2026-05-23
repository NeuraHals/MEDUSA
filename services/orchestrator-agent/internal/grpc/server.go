package grpc

import (
	"context"
	"net"

	"github.com/antigravity/mono/services/orchestrator-agent/internal/state"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ExecutionServer exposes the gRPC ExecutionService for the AOA.
type ExecutionServer struct {
	log    *zap.Logger
	redis  *state.RedisClient
	server *googlegrpc.Server
}

func NewExecutionServer(log *zap.Logger, redis *state.RedisClient) *ExecutionServer {
	s := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(loggingInterceptor(log)),
	)
	reflection.Register(s)
	return &ExecutionServer{log: log, redis: redis, server: s}
}

func (s *ExecutionServer) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.log.Info("gRPC server listening", zap.String("addr", addr))
	return s.server.Serve(lis)
}

func (s *ExecutionServer) GracefulStop() {
	s.server.GracefulStop()
}

// GetExecutionState returns the current state machine state for a blueprint.
// Used by the HOA and AOA dashboard for real-time monitoring.
func (s *ExecutionServer) GetExecutionState(ctx context.Context, blueprintID string) (string, error) {
	return s.redis.GetExecutionState(ctx, blueprintID)
}

func loggingInterceptor(log *zap.Logger) googlegrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *googlegrpc.UnaryServerInfo,
		handler googlegrpc.UnaryHandler,
	) (interface{}, error) {
		log.Info("gRPC request", zap.String("method", info.FullMethod))
		resp, err := handler(ctx, req)
		if err != nil {
			log.Error("gRPC error", zap.String("method", info.FullMethod), zap.Error(err))
		}
		return resp, err
	}
}

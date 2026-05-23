package grpc

import (
	"context"
	"net"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/approval"
	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/antigravity/mono/services/oversight-agent/internal/security"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// ApprovalServer implements the gRPC ApprovalService.
type ApprovalServer struct {
	log     *zap.Logger
	svc     *approval.Service
	server  *googlegrpc.Server
}

func NewApprovalServer(log *zap.Logger, svc *approval.Service) *ApprovalServer {
	s := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(chainInterceptors(
			loggingInterceptor(log),
			spiffeInterceptor(log),
		)),
	)
	reflection.Register(s)
	return &ApprovalServer{log: log, svc: svc, server: s}
}

func (s *ApprovalServer) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.log.Info("HOA gRPC listening", zap.String("addr", addr))
	return s.server.Serve(lis)
}

func (s *ApprovalServer) GracefulStop() {
	s.server.GracefulStop()
}

// RequestApproval is the gRPC handler called by the AOA for Tier 2 approvals.
func (s *ApprovalServer) RequestApproval(ctx context.Context, req *models.ApprovalRequest) (*models.ApprovalRecord, error) {
	if req.BlueprintID == "" || req.ActionID == "" || req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "blueprint_id, action_id, and idempotency_key are required")
	}

	// Validate approval token (biometric JWT verification stub)
	if !security.ValidateApprovalToken(req.IdempotencyKey) {
		return nil, status.Error(codes.Unauthenticated, "invalid approval token")
	}

	ctx, cancel := context.WithTimeout(ctx, 70*time.Second) // 60s + 10s buffer
	defer cancel()

	record, err := s.svc.RequestApproval(ctx, req)
	if err != nil {
		s.log.Error("approval service error", zap.Error(err),
			zap.String("blueprint_id", req.BlueprintID))
		return nil, status.Errorf(codes.Internal, "approval failed: %v", err)
	}

	return record, nil
}

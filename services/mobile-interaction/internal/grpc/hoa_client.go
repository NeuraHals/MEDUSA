package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/mobile-interaction/internal/models"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// HOAClient is a gRPC client for forwarding biometric approval decisions to the HOA.
type HOAClient struct {
	log      *zap.Logger
	endpoint string
	conn     *googlegrpc.ClientConn
}

func NewHOAClient(log *zap.Logger, endpoint string) (*HOAClient, error) {
	conn, err := googlegrpc.Dial(endpoint,
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		googlegrpc.WithBlock(),
		googlegrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("HOA gRPC dial failed: %w", err)
	}
	return &HOAClient{log: log, endpoint: endpoint, conn: conn}, nil
}

// ForwardApproval relays a biometric approval decision to the HOA ApprovalService.
// In production: uses generated tonic/protobuf stubs from api/proto/orchestration/v1.
// This stub performs a raw gRPC invocation as a placeholder.
func (c *HOAClient) ForwardApproval(ctx context.Context, req *models.ApprovalRelayEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c.log.Info("forwarding approval to HOA",
		zap.String("blueprint_id", req.BlueprintID),
		zap.String("action_id", req.ActionID),
		zap.String("decision", req.Decision),
		zap.String("operator_id", req.OperatorID),
	)

	// Full implementation: call pb.NewApprovalServiceClient(c.conn).RequestApproval(ctx, &pb.ApprovalRequest{...})
	// Using conn.Invoke as a compile-safe placeholder.
	err := c.conn.Invoke(ctx, "/antigravity.orchestration.v1.ApprovalService/RequestApproval", req, nil)
	if err != nil {
		c.log.Error("HOA forward failed", zap.Error(err), zap.String("blueprint_id", req.BlueprintID))
		return fmt.Errorf("HOA gRPC forward failed: %w", err)
	}

	return nil
}

func (c *HOAClient) Close() { _ = c.conn.Close() }

package approval

import (
	"context"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/antigravity/mono/services/oversight-agent/internal/state"
	"go.uber.org/zap"
)

// GlassBreakHandler processes timeout events and autonomously approves blocked Tier 2 actions.
// A Glass Break override is an immutable, audited, autonomous override of a human approval requirement.
type GlassBreakHandler struct {
	log      *zap.Logger
	redis    *state.RedisClient
	notifyCh chan *models.ApprovalRecord
}

func NewGlassBreakHandler(log *zap.Logger, redis *state.RedisClient) *GlassBreakHandler {
	return &GlassBreakHandler{
		log:      log,
		redis:    redis,
		notifyCh: make(chan *models.ApprovalRecord, 32),
	}
}

// Execute processes a Glass Break override for the given blueprint.
// It updates the approval record to GLASS_BREAK and emits the record for audit publishing.
func (g *GlassBreakHandler) Execute(ctx context.Context, blueprintID string) (*models.ApprovalRecord, error) {
	record, err := g.redis.GetApprovalRecord(ctx, blueprintID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		g.log.Error("Glass Break attempted on non-existent record",
			zap.String("blueprint_id", blueprintID),
		)
		return nil, nil
	}

	// Only trigger Glass Break on PENDING records
	if record.Status != models.StatusPending {
		g.log.Info("Glass Break skipped — approval already resolved",
			zap.String("blueprint_id", blueprintID),
			zap.String("status", string(record.Status)),
		)
		return record, nil
	}

	now := time.Now().UTC()
	record.Status = models.StatusGlassBreak
	record.GlassBreakUsed = true
	record.ProcessedAt = &now
	record.ApproverID = "SYSTEM:GLASS_BREAK"

	if err := g.redis.StoreApprovalRecord(ctx, record); err != nil {
		return nil, err
	}

	g.log.Warn("Glass Break override activated",
		zap.String("blueprint_id", blueprintID),
		zap.String("hospital_id", record.Request.HospitalID),
		zap.Uint32("pri_score", record.Request.PRIScore),
		zap.String("target_api", record.Request.TargetAPI),
	)

	// Emit for audit trail publication
	select {
	case g.notifyCh <- record:
	default:
		g.log.Error("Glass Break notify channel full",
			zap.String("blueprint_id", blueprintID),
		)
	}

	return record, nil
}

// NotifyCh exposes the Glass Break trigger channel for the approval service.
func (g *GlassBreakHandler) NotifyCh() <-chan *models.ApprovalRecord {
	return g.notifyCh
}

package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/antigravity/mono/services/oversight-agent/internal/state"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("oversight-agent/approval")

// Store provides CRUD operations for approval records.
type Store struct {
	log   *zap.Logger
	redis *state.RedisClient
}

func NewStore(log *zap.Logger, redis *state.RedisClient) *Store {
	return &Store{log: log, redis: redis}
}

// Create stores a new approval record with PENDING status and returns it.
func (s *Store) Create(ctx context.Context, req *models.ApprovalRequest, timeoutSecs int) (*models.ApprovalRecord, error) {
	ctx, span := tracer.Start(ctx, "store.create")
	defer span.End()
	span.SetAttributes(
		attribute.String("blueprint.id", req.BlueprintID),
		attribute.String("hospital.id", req.HospitalID),
		attribute.Int("pri.score", int(req.PRIScore)),
	)
	now := time.Now().UTC()
	record := &models.ApprovalRecord{
		RecordID:    uuid.New().String(),
		Request:     *req,
		Status:      models.StatusPending,
		RequestedAt: now,
		ExpiresAt:   now.Add(time.Duration(timeoutSecs) * time.Second),
	}
	if err := s.redis.StoreApprovalRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("store create failed: %w", err)
	}
	s.log.Info("approval record created",
		zap.String("record_id", record.RecordID),
		zap.String("blueprint_id", req.BlueprintID),
		zap.String("expires_at", record.ExpiresAt.Format(time.RFC3339)),
	)
	return record, nil
}

// Get retrieves an approval record by blueprint ID.
func (s *Store) Get(ctx context.Context, blueprintID string) (*models.ApprovalRecord, error) {
	return s.redis.GetApprovalRecord(ctx, blueprintID)
}

// Resolve updates an approval record with a final decision.
func (s *Store) Resolve(
	ctx context.Context,
	blueprintID string,
	status models.ApprovalStatus,
	approverID string,
	biometricJWT string,
) (*models.ApprovalRecord, error) {
	ctx, span := tracer.Start(ctx, "store.resolve")
	defer span.End()
	record, err := s.redis.GetApprovalRecord(ctx, blueprintID)
	if err != nil || record == nil {
		return nil, fmt.Errorf("record not found: %s", blueprintID)
	}
	if record.Status != models.StatusPending {
		s.log.Warn("resolve attempted on non-pending record",
			zap.String("blueprint_id", blueprintID),
			zap.String("current_status", string(record.Status)),
		)
		return record, nil
	}
	now := time.Now().UTC()
	record.Status = status
	record.ApproverID = approverID
	record.BiometricJWT = biometricJWT
	record.ProcessedAt = &now
	if err := s.redis.StoreApprovalRecord(ctx, record); err != nil {
		return nil, err
	}
	s.log.Info("approval record resolved",
		zap.String("blueprint_id", blueprintID),
		zap.String("status", string(status)),
		zap.String("approver_id", approverID),
	)
	return record, nil
}

// BuildDecisionEvent creates an ApprovalDecisionEvent from a resolved record.
func BuildDecisionEvent(record *models.ApprovalRecord, schemaVersion string) *models.ApprovalDecisionEvent {
	return &models.ApprovalDecisionEvent{
		EventID:        uuid.New().String(),
		BlueprintID:    record.Request.BlueprintID,
		ActionID:       record.Request.ActionID,
		HospitalID:     record.Request.HospitalID,
		Status:         record.Status,
		ApproverID:     record.ApproverID,
		GlassBreakUsed: record.GlassBreakUsed,
		TraceID:        record.Request.TraceID,
		IdempotencyKey: record.Request.IdempotencyKey,
		SchemaVersion:  schemaVersion,
		DecidedAt:      time.Now().UTC(),
	}
}

func Serialise(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

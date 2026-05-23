package approval

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/kafka"
	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/antigravity/mono/services/oversight-agent/internal/state"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service is the core approval orchestrator for the HOA.
// It coordinates the Store, TimeoutManager, GlassBreakHandler, and Kafka publisher.
type Service struct {
	log        *zap.Logger
	store      *Store
	timeout    *TimeoutManager
	glassBreak *GlassBreakHandler
	producer   *kafka.Producer
	redis      *state.RedisClient
	timeoutSecs int
	schemaVersion string
}

func NewService(
	log *zap.Logger,
	redis *state.RedisClient,
	producer *kafka.Producer,
	timeoutSecs int,
	schemaVersion string,
) *Service {
	store := NewStore(log, redis)
	timeout := NewTimeoutManager(log, redis)
	gb := NewGlassBreakHandler(log, redis)

	svc := &Service{
		log:           log,
		store:         store,
		timeout:       timeout,
		glassBreak:    gb,
		producer:      producer,
		redis:         redis,
		timeoutSecs:   timeoutSecs,
		schemaVersion: schemaVersion,
	}
	go svc.consumeTimeouts(context.Background())
	go svc.consumeGlassBreaks(context.Background())
	return svc
}

// RequestApproval is the primary entry point from the AOA gRPC call.
// It creates a pending record, starts the Glass Break timer, and blocks
// until the approval is resolved (approved, denied, or timed out).
func (s *Service) RequestApproval(ctx context.Context, req *models.ApprovalRequest) (*models.ApprovalRecord, error) {
	// Idempotency guard
	if s.redis.IsApprovalDuplicate(ctx, req.IdempotencyKey) {
		s.log.Info("duplicate approval request — returning cached result",
			zap.String("idempotency_key", req.IdempotencyKey),
		)
		record, _ := s.redis.GetApprovalRecord(ctx, req.BlueprintID)
		return record, nil
	}

	record, err := s.store.Create(ctx, req, s.timeoutSecs)
	if err != nil {
		return nil, fmt.Errorf("approval store create failed: %w", err)
	}

	// Start Glass Break countdown
	s.timeout.Schedule(ctx, req.BlueprintID)

	// Poll Redis for resolution (max timeoutSecs + 5s buffer)
	deadline := time.Now().Add(time.Duration(s.timeoutSecs+5) * time.Second)
	for time.Now().Before(deadline) {
		current, err := s.redis.GetApprovalRecord(ctx, req.BlueprintID)
		if err != nil {
			return nil, err
		}
		if current != nil && current.Status != models.StatusPending {
			_ = s.redis.MarkApprovalProcessed(ctx, req.IdempotencyKey)
			return current, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	// Timed out waiting — return TIMEOUT record
	now := time.Now().UTC()
	record.Status = models.StatusTimeout
	record.ProcessedAt = &now
	_ = s.redis.StoreApprovalRecord(ctx, record)
	_ = s.redis.MarkApprovalProcessed(ctx, req.IdempotencyKey)

	// Publish timeout event
	event := BuildDecisionEvent(record, s.schemaVersion)
	_ = s.producer.PublishDecision(ctx, event, req.BlueprintID, req.TraceID, req.IdempotencyKey)

	return record, nil
}

// consumeTimeouts listens for Glass Break timeout signals and executes the override.
func (s *Service) consumeTimeouts(ctx context.Context) {
	for {
		select {
		case blueprintID := <-s.timeout.TimeoutCh():
			resolved, err := s.glassBreak.Execute(ctx, blueprintID)
			if err != nil {
				s.log.Error("glass break execution failed",
					zap.String("blueprint_id", blueprintID),
					zap.Error(err),
				)
				continue
			}
			if resolved != nil {
				event := BuildDecisionEvent(resolved, s.schemaVersion)
				if err := s.producer.PublishDecision(ctx, event, blueprintID,
					resolved.Request.TraceID, resolved.Request.IdempotencyKey); err != nil {
					s.log.Error("glass break event publish failed", zap.Error(err))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// consumeGlassBreaks listens for autonomous Glass Break records for audit publishing.
func (s *Service) consumeGlassBreaks(ctx context.Context) {
	for {
		select {
		case record := <-s.glassBreak.NotifyCh():
			event := BuildDecisionEvent(record, s.schemaVersion)
			event.EventID = uuid.New().String() // ensure unique audit event
			if err := s.producer.PublishDecision(ctx, event,
				record.Request.BlueprintID, record.Request.TraceID, record.Request.IdempotencyKey); err != nil {
				s.log.Error("glass break audit publish failed", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

// Stop shuts down the timeout manager.
func (s *Service) Stop() {
	s.timeout.Stop()
}

// GetStore exposes the approval store for HTTP handlers.
func (s *Service) GetStore() *Store {
	return s.store
}

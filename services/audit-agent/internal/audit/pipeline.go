package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/hash"
	"github.com/antigravity/mono/services/audit-agent/internal/models"
	"github.com/antigravity/mono/services/audit-agent/internal/replay"
	"github.com/antigravity/mono/services/audit-agent/internal/retention"
	"github.com/antigravity/mono/services/audit-agent/internal/state"
	"github.com/antigravity/mono/services/audit-agent/internal/storage"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("audit-agent/pipeline")

// Pipeline is the core audit processing engine.
// For each inbound Kafka event it:
// 1. Classifies the retention class
// 2. Retrieves the chain head and links the new event
// 3. Computes the SHA-256 chain hash
// 4. Persists immutably to S3 WORM with Object Lock
// 5. Writes the replay index entry
// 6. Updates the chain head in Redis
type Pipeline struct {
	log      *zap.Logger
	redis    *state.RedisClient
	worm     *storage.WORMClient
	indexer  *replay.Indexer
	policy   *retention.Policy
	schemaVer string
}

func NewPipeline(
	log *zap.Logger,
	redis *state.RedisClient,
	worm *storage.WORMClient,
	indexer *replay.Indexer,
	policy *retention.Policy,
	schemaVer string,
) *Pipeline {
	return &Pipeline{log: log, redis: redis, worm: worm, indexer: indexer, policy: policy, schemaVer: schemaVer}
}

// Process ingests a raw Kafka message payload and runs the full audit pipeline.
func (p *Pipeline) Process(ctx context.Context, rawPayload []byte, traceID, agentID string) error {
	ctx, span := tracer.Start(ctx, "pipeline.process")
	defer span.End()

	// Deserialise to a generic envelope — we accept any orchestration event
	var envelope map[string]interface{}
	if err := json.Unmarshal(rawPayload, &envelope); err != nil {
		return fmt.Errorf("envelope deserialise failed: %w", err)
	}

	// Extract mandatory fields from the envelope
	eventID := stringField(envelope, "event_id", "blueprint_id", "result_id", "request_id")
	if eventID == "" {
		eventID = uuid.New().String() // fallback: generate a stable ID
	}

	// Idempotency check — skip already-audited events
	if p.redis.IsAlreadyAudited(ctx, eventID) {
		p.log.Info("idempotency hit — audit already persisted", zap.String("event_id", eventID))
		return nil
	}

	crisisID  := stringField(envelope, "crisis_id")
	hospitalID := stringField(envelope, "hospital_id")
	agentIDField := stringField(envelope, "agent_id")
	if agentIDField == "" { agentIDField = agentID }
	blueprintID := stringField(envelope, "blueprint_id")
	severity := stringField(envelope, "severity")
	occurredAt := time.Now().UTC()

	// Detect event type from envelope structure
	eventType := detectEventType(envelope)

	// Classify retention
	legalHold := retention.LegalHoldActive(crisisID)
	retClass := retention.ClassifyRetention(eventType, severity, legalHold)

	// Retrieve current chain head (previous hash)
	previousHash, err := p.redis.GetChainHead(ctx, crisisID)
	if err != nil {
		p.log.Warn("chain head retrieval failed — using genesis", zap.Error(err))
		previousHash = ""
	}
	if previousHash == "" {
		previousHash = hash.GenesisHash
	}

	// Compute event hash
	hashInput := hash.EventHashInput{
		EventID:      eventID,
		EventType:    string(eventType),
		CrisisID:     crisisID,
		BlueprintID:  blueprintID,
		HospitalID:   hospitalID,
		AgentID:      agentIDField,
		PreviousHash: previousHash,
		OccurredAt:   occurredAt,
		Payload:      envelope,
	}
	eventHash, err := hash.ComputeEventHash(hashInput)
	if err != nil {
		return fmt.Errorf("hash computation failed: %w", err)
	}

	span.SetAttributes(
		attribute.String("event.id", eventID),
		attribute.String("event.type", string(eventType)),
		attribute.String("crisis.id", crisisID),
		attribute.String("event.hash", eventHash),
	)

	// Build the canonical audit event record
	auditEvent := &models.AuditEvent{
		EventID:        eventID,
		EventType:      eventType,
		CrisisID:       crisisID,
		BlueprintID:    blueprintID,
		HospitalID:     hospitalID,
		AgentID:        agentIDField,
		Payload:        envelope,
		PreviousHash:   previousHash,
		EventHash:      eventHash,
		TraceID:        traceID,
		IdempotencyKey: eventID,
		SchemaVersion:  p.schemaVer,
		Severity:       severity,
		LegalHold:      legalHold,
		RetentionClass: retClass,
		OccurredAt:     occurredAt,
		PersistedAt:    time.Now().UTC(),
	}

	retainUntil := p.policy.RetainUntil(auditEvent)

	// Persist to WORM — this is the immutable record of truth
	s3Key, err := p.worm.PersistEvent(ctx, auditEvent, retainUntil)
	if err != nil {
		return fmt.Errorf("WORM persist failed: %w", err)
	}

	// Update replay index
	if err := p.indexer.Index(ctx, auditEvent, s3Key); err != nil {
		p.log.Warn("replay index update failed — WORM persist succeeded",
			zap.Error(err), zap.String("event_id", eventID),
		)
	}

	// Advance chain head atomically
	if err := p.redis.SetChainHead(ctx, crisisID, eventHash); err != nil {
		p.log.Warn("chain head update failed", zap.Error(err), zap.String("crisis_id", crisisID))
	}
	chainLen, _ := p.redis.IncrChainLength(ctx, crisisID)

	// Mark as audited to prevent replay double-processing
	_ = p.redis.MarkAudited(ctx, eventID)

	p.log.Info("audit event persisted",
		zap.String("event_id", eventID),
		zap.String("event_type", string(eventType)),
		zap.String("s3_key", s3Key),
		zap.String("event_hash", eventHash),
		zap.Int64("chain_length", chainLen),
		zap.String("retention_class", string(retClass)),
		zap.Bool("legal_hold", legalHold),
	)

	return nil
}

// detectEventType infers the AuditEventType from the envelope field signatures.
func detectEventType(envelope map[string]interface{}) models.AuditEventType {
	if _, ok := envelope["glass_break_used"]; ok {
		if v, ok := envelope["glass_break_used"].(bool); ok && v {
			return models.EventGlassBreak
		}
	}
	if v, ok := envelope["decision"].(string); ok {
		if v == "DENIED" { return models.EventDenial }
		if v == "APPROVED" { return models.EventApproval }
	}
	if _, ok := envelope["undo_actions"]; ok { return models.EventRollback }
	if _, ok := envelope["final_confidence"]; ok { return models.EventCorrelation }
	if _, ok := envelope["blueprint_id"]; ok {
		if _, hasActions := envelope["actions"]; hasActions { return models.EventAllocation }
		return models.EventExecution
	}
	if _, ok := envelope["confidence_score"]; ok { return models.EventSignalIngest }
	return models.EventExecution
}

// stringField returns the first non-empty string value from the given keys.
func stringField(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

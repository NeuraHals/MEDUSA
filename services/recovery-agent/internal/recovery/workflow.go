package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/recovery-agent/internal/graph"
	"github.com/antigravity/mono/services/recovery-agent/internal/middleware"
	"github.com/antigravity/mono/services/recovery-agent/internal/models"
	"github.com/antigravity/mono/services/recovery-agent/internal/rollback"
	"github.com/antigravity/mono/services/recovery-agent/internal/state"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("recovery-agent/workflow")

// Workflow orchestrates the full rollback lifecycle:
// RECEIVED → PLANNING → EXECUTING → COMPLETED | FAILED | PARTIAL
type Workflow struct {
	log          *zap.Logger
	redis        *state.RedisClient
	executor     *rollback.Executor
	maxRetries   int
	degradedMode bool
	schemaVer    string
	cb           *middleware.CircuitBreaker
}

func NewWorkflow(
	log *zap.Logger,
	redis *state.RedisClient,
	executor *rollback.Executor,
	maxRetries int,
	degradedMode bool,
	schemaVer string,
	cbFailureThreshold, cbRecoverySecs int,
) *Workflow {
	return &Workflow{
		log:          log,
		redis:        redis,
		executor:     executor,
		maxRetries:   maxRetries,
		degradedMode: degradedMode,
		schemaVer:    schemaVer,
		cb:           middleware.NewCircuitBreaker(cbFailureThreshold, cbRecoverySecs),
	}
}

// Execute runs the full rollback workflow for a RollbackManifest.
// Returns a RecoveryEvent representing the final outcome.
func (w *Workflow) Execute(ctx context.Context, manifest *models.RollbackManifest) *models.RecoveryEvent {
	ctx, span := tracer.Start(ctx, "workflow.execute")
	defer span.End()
	span.SetAttributes(
		attribute.String("rollback.id", manifest.RollbackID),
		attribute.String("blueprint.id", manifest.BlueprintID),
		attribute.Int("undo_action.count", len(manifest.UndoActions)),
	)

	// Persist initial record
	record := &models.RollbackRecord{
		RollbackID:     manifest.RollbackID,
		BlueprintID:    manifest.BlueprintID,
		CrisisID:       manifest.CrisisID,
		HospitalID:     manifest.HospitalID,
		State:          models.RollbackReceived,
		Reason:         manifest.Reason,
		TraceID:        manifest.TraceID,
		IdempotencyKey: manifest.IdempotencyKey,
		StartedAt:      time.Now().UTC(),
	}
	_ = w.redis.StoreRecord(ctx, record)
	_ = w.redis.SetState(ctx, manifest.RollbackID, models.RollbackPlanning)

	// PLANNING: compute dependency-aware execution order
	orderedActions, err := w.planOrder(manifest.UndoActions)
	if err != nil {
		w.log.Error("dependency graph planning failed",
			zap.Error(err), zap.String("rollback_id", manifest.RollbackID),
		)
		_ = w.redis.SetState(ctx, manifest.RollbackID, models.RollbackFailed)
		return w.buildEvent(manifest, models.RollbackFailed, nil)
	}

	_ = w.redis.SetState(ctx, manifest.RollbackID, models.RollbackExecuting)

	// EXECUTING: run undo actions in dependency order with retry and circuit breaking
	results := make([]models.ActionResult, 0, len(orderedActions))
	anyFailed := false

	for _, action := range orderedActions {
		if w.cb.IsOpen() {
			w.log.Warn("circuit breaker open — degraded-mode continuation",
				zap.String("action_id", action.ActionID),
			)
			results = append(results, models.ActionResult{
				ActionID:   action.ActionID,
				ResourceID: action.ResourceID,
				Success:    false,
				ErrorCode:  "CIRCUIT_OPEN",
			})
			anyFailed = true
			continue
		}

		var result models.ActionResult
		var attempts int

		err := middleware.Retry(ctx, w.log, w.maxRetries, func() error {
			attempts++
			result = w.executor.Execute(ctx, action, manifest.TraceID, attempts)
			if !result.Success {
				return fmt.Errorf("undo failed: %s", result.ErrorCode)
			}
			return nil
		})

		result.Attempts = attempts
		results = append(results, result)

		if err != nil {
			w.cb.RecordFailure()
			anyFailed = true
			w.log.Error("undo action failed after retries",
				zap.String("action_id", action.ActionID),
				zap.String("rollback_id", manifest.RollbackID),
				zap.Error(err),
			)
			if !w.degradedMode {
				// Stop-on-first-failure for non-degraded mode
				break
			}
			// Degraded mode: continue attempting remaining actions
		} else {
			w.cb.RecordSuccess()
		}
	}

	// Determine final state
	finalState := models.RollbackCompleted
	if anyFailed {
		successCount := 0
		for _, r := range results {
			if r.Success { successCount++ }
		}
		if successCount == 0 {
			finalState = models.RollbackFailed
		} else {
			finalState = models.RollbackPartial
		}
		if w.degradedMode {
			finalState = models.RollbackDegraded
		}
	}

	record.ActionResults = results
	_ = w.redis.StoreRecord(ctx, record)
	_ = w.redis.SetState(ctx, manifest.RollbackID, finalState)

	w.log.Info("rollback workflow complete",
		zap.String("rollback_id", manifest.RollbackID),
		zap.String("state", string(finalState)),
		zap.Int("actions_attempted", len(results)),
	)

	return w.buildEvent(manifest, finalState, results)
}

func (w *Workflow) planOrder(actions []models.UndoAction) ([]models.UndoAction, error) {
	if len(actions) == 0 {
		return actions, nil
	}
	// Check if any action has dependencies; if not, return as-is (AOA LIFO order)
	hasDeps := false
	for _, a := range actions {
		if len(a.DependsOn) > 0 {
			hasDeps = true
			break
		}
	}
	if !hasDeps {
		return actions, nil
	}
	g, err := graph.Build(actions)
	if err != nil {
		return nil, err
	}
	return g.TopologicalOrder()
}

func (w *Workflow) buildEvent(manifest *models.RollbackManifest, state models.RollbackState, results []models.ActionResult) *models.RecoveryEvent {
	if results == nil {
		results = []models.ActionResult{}
	}
	return &models.RecoveryEvent{
		EventID:        uuid.New().String(),
		RollbackID:     manifest.RollbackID,
		BlueprintID:    manifest.BlueprintID,
		CrisisID:       manifest.CrisisID,
		HospitalID:     manifest.HospitalID,
		State:          state,
		ActionResults:  results,
		TraceID:        manifest.TraceID,
		IdempotencyKey: manifest.IdempotencyKey,
		SchemaVersion:  w.schemaVer,
		CompletedAt:    time.Now().UTC(),
	}
}

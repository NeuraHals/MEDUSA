package rollback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/antigravity/mono/services/recovery-agent/internal/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var execTracer = otel.Tracer("recovery-agent/executor")

// Executor dispatches individual undo actions against their target APIs.
type Executor struct {
	log            *zap.Logger
	httpClient     *http.Client
	undoTimeoutSecs int
}

func NewExecutor(log *zap.Logger, undoTimeoutSecs int) *Executor {
	return &Executor{
		log:             log,
		undoTimeoutSecs: undoTimeoutSecs,
		httpClient:      &http.Client{Timeout: time.Duration(undoTimeoutSecs) * time.Second},
	}
}

// Execute dispatches a single undo action and returns its result.
func (e *Executor) Execute(ctx context.Context, action models.UndoAction, traceID string, attempt int) models.ActionResult {
	ctx, span := execTracer.Start(ctx, fmt.Sprintf("undo.%s", action.ActionID))
	defer span.End()
	span.SetAttributes(
		attribute.String("action.id", action.ActionID),
		attribute.String("resource.id", action.ResourceID),
		attribute.String("undo.api", action.UndoAPI),
	)

	body, err := json.Marshal(map[string]interface{}{
		"resource_id":     action.ResourceID,
		"resource_type":   action.ResourceType,
		"undo_parameters": action.UndoParameters,
		"trace_id":        traceID,
		"action_id":       action.ActionID,
	})
	if err != nil {
		return models.ActionResult{ActionID: action.ActionID, ResourceID: action.ResourceID, Success: false, ErrorCode: "SERIALISATION_ERROR", Attempts: attempt}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, action.UndoAPI, bytes.NewReader(body))
	if err != nil {
		return models.ActionResult{ActionID: action.ActionID, ResourceID: action.ResourceID, Success: false, ErrorCode: "REQUEST_BUILD_ERROR", Attempts: attempt}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-ID", traceID)
	req.Header.Set("X-Action-ID", action.ActionID)

	e.log.Info("dispatching undo action",
		zap.String("action_id", action.ActionID),
		zap.String("undo_api", action.UndoAPI),
		zap.Int("attempt", attempt),
	)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return models.ActionResult{ActionID: action.ActionID, ResourceID: action.ResourceID, Success: false, ErrorCode: fmt.Sprintf("HTTP_ERR: %v", err), Attempts: attempt}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.ActionResult{ActionID: action.ActionID, ResourceID: action.ResourceID, Success: false, ErrorCode: fmt.Sprintf("HTTP_%d", resp.StatusCode), Attempts: attempt}
	}
	return models.ActionResult{ActionID: action.ActionID, ResourceID: action.ResourceID, Success: true, Attempts: attempt}
}

// ExecuteParallel runs independent undo actions concurrently.
func (e *Executor) ExecuteParallel(ctx context.Context, actions []models.UndoAction, traceID string) []models.ActionResult {
	results := make([]models.ActionResult, len(actions))
	var wg sync.WaitGroup
	for i, action := range actions {
		i, action := i, action
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = e.Execute(ctx, action, traceID, 1)
		}()
	}
	wg.Wait()
	return results
}

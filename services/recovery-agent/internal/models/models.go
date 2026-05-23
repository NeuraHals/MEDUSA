package models

import "time"

// RollbackManifest is the inbound event from the AOA or RAA describing
// a failed blueprint that requires recovery and resource reversal.
type RollbackManifest struct {
	RollbackID     string       `json:"rollback_id"`
	BlueprintID    string       `json:"blueprint_id"`
	CrisisID       string       `json:"crisis_id"`
	HospitalID     string       `json:"hospital_id"`
	Reason         string       `json:"reason"`
	UndoActions    []UndoAction `json:"undo_actions"` // LIFO-ordered by AOA
	TraceID        string       `json:"trace_id"`
	IdempotencyKey string       `json:"idempotency_key"`
	SchemaVersion  string       `json:"schema_version"`
	CreatedAt      time.Time    `json:"created_at"`
}

// UndoAction is a single reversal step within a rollback manifest.
type UndoAction struct {
	ActionID       string                 `json:"action_id"`
	ResourceID     string                 `json:"resource_id"`
	ResourceType   string                 `json:"resource_type"`
	UndoAPI        string                 `json:"undo_api"`
	UndoParameters map[string]interface{} `json:"undo_parameters"`
	DependsOn      []string               `json:"depends_on,omitempty"` // action_ids that must complete first
	Priority       int                    `json:"priority"`             // lower = executed first
}

// RollbackState is the persistent state of a rollback workflow.
type RollbackState string

const (
	RollbackReceived   RollbackState = "RECEIVED"
	RollbackPlanning   RollbackState = "PLANNING"
	RollbackExecuting  RollbackState = "EXECUTING"
	RollbackCompleted  RollbackState = "COMPLETED"
	RollbackFailed     RollbackState = "FAILED"
	RollbackPartial    RollbackState = "PARTIAL"
	RollbackDegraded   RollbackState = "DEGRADED" // degraded-mode recovery path
)

// RollbackRecord tracks per-rollback execution state in Redis.
type RollbackRecord struct {
	RollbackID     string        `json:"rollback_id"`
	BlueprintID    string        `json:"blueprint_id"`
	CrisisID       string        `json:"crisis_id"`
	HospitalID     string        `json:"hospital_id"`
	State          RollbackState `json:"state"`
	ActionResults  []ActionResult `json:"action_results"`
	Reason         string        `json:"reason"`
	TraceID        string        `json:"trace_id"`
	IdempotencyKey string        `json:"idempotency_key"`
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty"`
}

// ActionResult records the outcome of a single undo action.
type ActionResult struct {
	ActionID  string `json:"action_id"`
	ResourceID string `json:"resource_id"`
	Success   bool   `json:"success"`
	ErrorCode string `json:"error_code,omitempty"`
	Attempts  int    `json:"attempts"`
}

// RecoveryEvent is published after rollback completion (success or failure).
type RecoveryEvent struct {
	EventID        string        `json:"event_id"`
	RollbackID     string        `json:"rollback_id"`
	BlueprintID    string        `json:"blueprint_id"`
	CrisisID       string        `json:"crisis_id"`
	HospitalID     string        `json:"hospital_id"`
	State          RollbackState `json:"state"`
	ActionResults  []ActionResult `json:"action_results"`
	TraceID        string        `json:"trace_id"`
	IdempotencyKey string        `json:"idempotency_key"`
	SchemaVersion  string        `json:"schema_version"`
	CompletedAt    time.Time     `json:"completed_at"`
}

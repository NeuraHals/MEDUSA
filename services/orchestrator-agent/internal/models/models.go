package models

import "time"

// AllocationBlueprint mirrors the RAA output schema exactly.
type AllocationBlueprint struct {
	BlueprintID    string             `json:"blueprint_id"`
	CrisisID       string             `json:"crisis_id"`
	HospitalID     string             `json:"hospital_id"`
	Classification string             `json:"classification"`
	Tier           uint8              `json:"tier"`
	PRIScore       uint32             `json:"pri_score"`
	Actions        []AllocationAction `json:"actions"`
	TraceID        string             `json:"trace_id"`
	IdempotencyKey string             `json:"idempotency_key"`
	SchemaVersion  string             `json:"schema_version"`
	GeneratedAt    time.Time          `json:"generated_at"`
}

type AllocationAction struct {
	ActionID        string                 `json:"action_id"`
	ActionType      string                 `json:"action_type"`
	ResourceID      string                 `json:"resource_id"`
	ResourceType    string                 `json:"resource_type"`
	TargetAPI       string                 `json:"target_api"`
	Parameters      map[string]interface{} `json:"parameters"`
	OpportunityCost float64                `json:"opportunity_cost"`
	Tier            uint8                  `json:"tier"`
}

// ExecutionEvent is the output event published after each action execution attempt.
type ExecutionEvent struct {
	EventID        string          `json:"event_id"`
	BlueprintID    string          `json:"blueprint_id"`
	CrisisID       string          `json:"crisis_id"`
	HospitalID     string          `json:"hospital_id"`
	State          ExecutionState  `json:"state"`
	ActionResults  []ActionResult  `json:"action_results"`
	TraceID        string          `json:"trace_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	SchemaVersion  string          `json:"schema_version"`
	ExecutedAt     time.Time       `json:"executed_at"`
}

type ActionResult struct {
	ActionID   string `json:"action_id"`
	ResourceID string `json:"resource_id"`
	Success    bool   `json:"success"`
	ErrorCode  string `json:"error_code,omitempty"`
}

// RollbackEvent is published when a blueprint must be reversed.
type RollbackEvent struct {
	RollbackID     string       `json:"rollback_id"`
	BlueprintID    string       `json:"blueprint_id"`
	CrisisID       string       `json:"crisis_id"`
	HospitalID     string       `json:"hospital_id"` // matches RAA RollbackManifest
	Reason         string       `json:"reason"`
	UndoActions    []UndoAction `json:"undo_actions"`
	TraceID        string       `json:"trace_id"`
	IdempotencyKey string       `json:"idempotency_key"`
	SchemaVersion  string       `json:"schema_version"`
	CreatedAt      time.Time    `json:"created_at"`
}

type UndoAction struct {
	ActionID       string                 `json:"action_id"`
	ResourceID     string                 `json:"resource_id"`
	UndoAPI        string                 `json:"undo_api"`
	UndoParameters map[string]interface{} `json:"undo_parameters"`
}

// ExecutionState represents the AOA state machine states.
type ExecutionState string

const (
	StateReceived    ExecutionState = "RECEIVED"
	StateValidating  ExecutionState = "VALIDATING"
	StateApproved    ExecutionState = "APPROVED"
	StateExecuting   ExecutionState = "EXECUTING"
	StateExecuted    ExecutionState = "EXECUTED"
	StateFailed      ExecutionState = "FAILED"
	StateRollingBack ExecutionState = "ROLLING_BACK"
	StateRolledBack  ExecutionState = "ROLLED_BACK"
)

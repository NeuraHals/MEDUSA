package models

import "time"

// ApprovalRequest is the inbound request from the AOA via gRPC.
type ApprovalRequest struct {
	BlueprintID    string `json:"blueprint_id"`
	ActionID       string `json:"action_id"`
	TargetAPI      string `json:"target_api"`
	PRIScore       uint32 `json:"pri_score"`
	TraceID        string `json:"trace_id"`
	IdempotencyKey string `json:"idempotency_key"`
	HospitalID     string `json:"hospital_id"`
}

// ApprovalRecord is the internal state stored in Redis for a pending approval.
type ApprovalRecord struct {
	RecordID       string         `json:"record_id"`
	Request        ApprovalRequest `json:"request"`
	Status         ApprovalStatus `json:"status"`
	RequestedAt    time.Time      `json:"requested_at"`
	ExpiresAt      time.Time      `json:"expires_at"`
	ProcessedAt    *time.Time     `json:"processed_at,omitempty"`
	ApproverID     string         `json:"approver_id,omitempty"`
	// Biometric JWT from Secure Enclave — nil until approved
	BiometricJWT   string         `json:"biometric_jwt,omitempty"`
	GlassBreakUsed bool           `json:"glass_break_used"`
}

// ApprovalStatus tracks the state of an approval record.
type ApprovalStatus string

const (
	StatusPending    ApprovalStatus = "PENDING"
	StatusApproved   ApprovalStatus = "APPROVED"
	StatusDenied     ApprovalStatus = "DENIED"
	StatusTimeout    ApprovalStatus = "TIMEOUT"
	StatusGlassBreak ApprovalStatus = "GLASS_BREAK"
)

// ApprovalDecisionEvent is published to Kafka after every approval resolution.
type ApprovalDecisionEvent struct {
	EventID        string         `json:"event_id"`
	BlueprintID    string         `json:"blueprint_id"`
	ActionID       string         `json:"action_id"`
	HospitalID     string         `json:"hospital_id"`
	Status         ApprovalStatus `json:"status"`
	ApproverID     string         `json:"approver_id,omitempty"`
	GlassBreakUsed bool           `json:"glass_break_used"`
	TraceID        string         `json:"trace_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	SchemaVersion  string         `json:"schema_version"`
	DecidedAt      time.Time      `json:"decided_at"`
}

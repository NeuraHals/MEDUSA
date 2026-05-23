package models

import "time"

// ApprovalDecision is the operator decision payload submitted via gRPC or HTTP.
type ApprovalDecision struct {
	ApprovalID    string    `json:"approval_id"`
	BlueprintID   string    `json:"blueprint_id"`
	Decision      string    `json:"decision"` // APPROVED | DENIED
	OperatorID    string    `json:"operator_id"`
	BiometricHash string    `json:"biometric_hash"`
	TraceID       string    `json:"trace_id"`
	DecidedAt     time.Time `json:"decided_at"`
}

// PendingApproval tracks in-flight approval state in Redis.
type PendingApproval struct {
	ApprovalID     string    `json:"approval_id"`
	BlueprintID    string    `json:"blueprint_id"`
	HospitalID     string    `json:"hospital_id"`
	PRIScore       uint32    `json:"pri_score"`
	Classification string    `json:"classification"`
	Description    string    `json:"description"`
	RequestingAgent string   `json:"requesting_agent"`
	TraceID        string    `json:"trace_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	State          string    `json:"state"` // PENDING | APPROVED | DENIED | GLASS_BREAK | TIMEOUT
	RequestedAt    time.Time `json:"requested_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

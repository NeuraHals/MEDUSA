package models

import "time"

// AuditEvent is the canonical audit record persisted immutably for every
// execution event, approval decision, and Glass Break override in the system.
type AuditEvent struct {
	EventID        string            `json:"event_id"`
	EventType      AuditEventType    `json:"event_type"`
	CrisisID       string            `json:"crisis_id"`
	BlueprintID    string            `json:"blueprint_id"`
	HospitalID     string            `json:"hospital_id"`
	OperatorID     string            `json:"operator_id,omitempty"`
	AgentID        string            `json:"agent_id"` // producing agent
	Payload        map[string]interface{} `json:"payload"`
	PreviousHash   string            `json:"previous_hash"` // chain-of-custody linkage
	EventHash      string            `json:"event_hash"`    // SHA-256(event content)
	TraceID        string            `json:"trace_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	SchemaVersion  string            `json:"schema_version"`
	Severity       string            `json:"severity"`
	LegalHold      bool              `json:"legal_hold"`
	RetentionClass RetentionClass    `json:"retention_class"`
	OccurredAt     time.Time         `json:"occurred_at"`
	PersistedAt    time.Time         `json:"persisted_at"`
}

// AuditEventType classifies what kind of system event is being recorded.
type AuditEventType string

const (
	EventExecution     AuditEventType = "EXECUTION"
	EventApproval      AuditEventType = "APPROVAL"
	EventGlassBreak    AuditEventType = "GLASS_BREAK"
	EventRollback      AuditEventType = "ROLLBACK"
	EventDenial        AuditEventType = "DENIAL"
	EventTimeout       AuditEventType = "TIMEOUT"
	EventCorrelation   AuditEventType = "CORRELATION"
	EventAllocation    AuditEventType = "ALLOCATION"
	EventSignalIngest  AuditEventType = "SIGNAL_INGEST"
)

// RetentionClass determines the minimum WORM retention period.
type RetentionClass string

const (
	RetentionStandard    RetentionClass = "STANDARD"    // 7 years — HIPAA minimum
	RetentionExtended    RetentionClass = "EXTENDED"    // 10 years — legal escalation
	RetentionLegalHold   RetentionClass = "LEGAL_HOLD"  // indefinite — do not purge
	RetentionForensic    RetentionClass = "FORENSIC"    // 25 years — criminal proceedings
)

// ReplayIndexEntry is stored in Redis to support fast event replay queries.
type ReplayIndexEntry struct {
	EventID     string         `json:"event_id"`
	EventType   AuditEventType `json:"event_type"`
	CrisisID    string         `json:"crisis_id"`
	BlueprintID string         `json:"blueprint_id"`
	HospitalID  string         `json:"hospital_id"`
	S3Key       string         `json:"s3_key"`
	EventHash   string         `json:"event_hash"`
	OccurredAt  time.Time      `json:"occurred_at"`
}

// ChainLink represents one link in the execution chain-of-custody ledger.
type ChainLink struct {
	LinkID       string    `json:"link_id"`
	CrisisID     string    `json:"crisis_id"`
	EventID      string    `json:"event_id"`
	EventType    string    `json:"event_type"`
	Hash         string    `json:"hash"`
	PreviousHash string    `json:"previous_hash"`
	ChainLength  int       `json:"chain_length"`
	CreatedAt    time.Time `json:"created_at"`
}

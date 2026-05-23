package models

import "time"

// UnifiedEvent is the canonical, normalized event schema published to Kafka.
// It maps directly to the JSON Schema registered in the Confluent Schema Registry.
type UnifiedEvent struct {
	EventID        string                 `json:"event_id"`
	SourceSystem   string                 `json:"source_system"`
	EventType      string                 `json:"event_type"`
	HospitalID     string                 `json:"hospital_id"`
	TimestampUTC   time.Time              `json:"timestamp_utc"`
	Payload        map[string]interface{} `json:"payload"`
	TraceID        string                 `json:"trace_id"`
	IdempotencyKey string                 `json:"idempotency_key"`
	SchemaVersion  string                 `json:"schema_version"`
	IngestedAt     time.Time              `json:"ingested_at"`
}

// TelemetryRequest is the inbound HTTP payload schema for POST /v1/telemetry.
type TelemetryRequest struct {
	SourceSystem string                 `json:"source_system"  validate:"required"`
	EventType    string                 `json:"event_type"     validate:"required"`
	HospitalID   string                 `json:"hospital_id"    validate:"required"`
	TimestampUTC string                 `json:"timestamp_utc"  validate:"required"`
	Payload      map[string]interface{} `json:"payload"`
}

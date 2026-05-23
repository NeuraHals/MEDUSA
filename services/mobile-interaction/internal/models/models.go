package models

import "time"

// ApprovalRequest is the inbound payload from a mobile operator via the HTTP API.
// The device submits this after a push notification wakes the app and the
// operator authenticates with biometrics.
type ApprovalRequest struct {
	BlueprintID    string `json:"blueprint_id"`
	ActionID       string `json:"action_id"`
	HospitalID     string `json:"hospital_id"`
	PRIScore       uint32 `json:"pri_score"`
	Decision       string `json:"decision"` // APPROVED | DENIED
	OperatorID     string `json:"operator_id"`
	BiometricJWT   string `json:"biometric_jwt"`
	IdempotencyKey string `json:"idempotency_key"`
	TraceID        string `json:"trace_id"`
	SchemaVersion  string `json:"schema_version"`
}

// PushNotificationRequest is the payload dispatched to the mobile device
// to wake it up and display the approval prompt.
type PushNotificationRequest struct {
	RequestID      string            `json:"request_id"`
	BlueprintID    string            `json:"blueprint_id"`
	ActionID       string            `json:"action_id"`
	HospitalID     string            `json:"hospital_id"`
	PRIScore       uint32            `json:"pri_score"`
	Classification string            `json:"classification"`
	Message        string            `json:"message"`
	OperatorID     string            `json:"operator_id"`
	DeviceToken    string            `json:"device_token"`
	Platform       string            `json:"platform"` // ios | android
	ExpiresAt      time.Time         `json:"expires_at"`
	TraceID        string            `json:"trace_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	SchemaVersion  string            `json:"schema_version"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// OfflineApprovalEntry is queued in Redis when the operator device is unreachable.
type OfflineApprovalEntry struct {
	EntryID        string    `json:"entry_id"`
	BlueprintID    string    `json:"blueprint_id"`
	ActionID       string    `json:"action_id"`
	HospitalID     string    `json:"hospital_id"`
	PRIScore       uint32    `json:"pri_score"`
	Classification string    `json:"classification"`
	Message        string    `json:"message"`
	OperatorID     string    `json:"operator_id"`
	TraceID        string    `json:"trace_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	QueuedAt       time.Time `json:"queued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// MobileSession tracks the push-state of an operator device.
type MobileSession struct {
	OperatorID  string    `json:"operator_id"`
	DeviceToken string    `json:"device_token"`
	Platform    string    `json:"platform"` // ios | android
	Online      bool      `json:"online"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	PhoneNumber string    `json:"phone_number,omitempty"` // for SMS fallback
}

// ApprovalRelayEvent is published to Kafka after an operator decision is received.
type ApprovalRelayEvent struct {
	EventID        string `json:"event_id"`
	BlueprintID    string `json:"blueprint_id"`
	ActionID       string `json:"action_id"`
	HospitalID     string `json:"hospital_id"`
	Decision       string `json:"decision"` // APPROVED | DENIED
	OperatorID     string `json:"operator_id"`
	BiometricJWT   string `json:"biometric_jwt"`
	TraceID        string `json:"trace_id"`
	IdempotencyKey string `json:"idempotency_key"`
	SchemaVersion  string `json:"schema_version"`
}

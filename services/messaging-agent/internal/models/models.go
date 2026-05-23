package models

import "time"

// NotificationRequest is the inbound Kafka event consumed from the execution bus.
// Published by the AOA after a blueprint execution event requiring stakeholder notification.
type NotificationRequest struct {
	RequestID      string            `json:"request_id"`
	CrisisID       string            `json:"crisis_id"`
	HospitalID     string            `json:"hospital_id"`
	Severity       string            `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Classification string            `json:"classification"`
	Message        string            `json:"message"`
	Channels       []Channel         `json:"channels"`
	Recipients     []Recipient       `json:"recipients"`
	TraceID        string            `json:"trace_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	SchemaVersion  string            `json:"schema_version"`
	CreatedAt      time.Time         `json:"created_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Channel specifies which messaging provider to use.
type Channel string

const (
	ChannelPagerDuty Channel = "PAGERDUTY"
	ChannelSMS       Channel = "SMS"
	ChannelAPNs      Channel = "APNS"
	ChannelFCM       Channel = "FCM"
	ChannelEmail     Channel = "EMAIL"
)

// Recipient holds delivery target metadata.
type Recipient struct {
	RecipientID  string  `json:"recipient_id"`
	Name         string  `json:"name"`
	Role         string  `json:"role"` // INCIDENT_COMMANDER, SOC, NOC, CLINICIAN
	PhoneNumber  string  `json:"phone_number,omitempty"`
	DeviceToken  string  `json:"device_token,omitempty"`
	PagerDutyKey string  `json:"pagerduty_key,omitempty"`
	Platform     string  `json:"platform,omitempty"` // ios, android
}

// DeliveryResult is the outcome of a single channel delivery attempt.
type DeliveryResult struct {
	RequestID      string    `json:"request_id"`
	RecipientID    string    `json:"recipient_id"`
	Channel        Channel   `json:"channel"`
	Success        bool      `json:"success"`
	ProviderRef    string    `json:"provider_ref,omitempty"`
	ErrorCode      string    `json:"error_code,omitempty"`
	Retries        int       `json:"retries"`
	TraceID        string    `json:"trace_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	SchemaVersion  string    `json:"schema_version"`
	DeliveredAt    time.Time `json:"delivered_at"`
}

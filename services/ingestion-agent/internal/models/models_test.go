package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/antigravity/mono/services/ingestion-agent/internal/models"
)

// TestUnifiedEventSerialisation asserts the UnifiedEvent round-trips through JSON cleanly.
func TestUnifiedEventSerialisation(t *testing.T) {
	event := models.UnifiedEvent{
		EventID:        "evt-001",
		SourceSystem:   "test-source",
		HospitalID:     "hospital-001",
		EventType:      "CRITICAL_ALERT",
		Payload:        map[string]interface{}{"key": "value"},
		TimestampUTC:   time.Now().UTC(),
		IdempotencyKey: "idem-001",
		SchemaVersion:  "1.0.0",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("serialisation failed: %v", err)
	}

	var decoded models.UnifiedEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("deserialisation failed: %v", err)
	}

	if decoded.EventID != event.EventID {
		t.Errorf("event_id mismatch: got %q, want %q", decoded.EventID, event.EventID)
	}
	if decoded.HospitalID != event.HospitalID {
		t.Errorf("hospital_id mismatch: got %q, want %q", decoded.HospitalID, event.HospitalID)
	}
}

// TestMandatoryFieldsValidation asserts required fields are validated.
func TestMandatoryFieldsValidation(t *testing.T) {
	event := models.UnifiedEvent{}
	if err := event.Validate(); err == nil {
		t.Fatal("empty event should fail validation")
	}
}

// TestValidEventPassesValidation asserts a complete event passes validation.
func TestValidEventPassesValidation(t *testing.T) {
	event := models.UnifiedEvent{
		EventID:      "evt-002",
		HospitalID:   "h-001",
		SourceSystem: "icu-monitor",
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid event failed validation: %v", err)
	}
}

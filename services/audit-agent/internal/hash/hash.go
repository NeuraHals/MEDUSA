package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// EventHashInput contains the canonical fields used to compute an event's SHA-256.
// Changing any field changes the hash — providing tamper evidence.
type EventHashInput struct {
	EventID      string
	EventType    string
	CrisisID     string
	BlueprintID  string
	HospitalID   string
	AgentID      string
	PreviousHash string
	OccurredAt   time.Time
	Payload      map[string]interface{}
}

// ComputeEventHash produces a deterministic SHA-256 digest of an audit event.
// The previous_hash field is included to create a chain of custody.
func ComputeEventHash(input EventHashInput) (string, error) {
	// Deterministic serialisation — sorted keys via json.Marshal on a struct
	canonical := struct {
		EventID      string                 `json:"event_id"`
		EventType    string                 `json:"event_type"`
		CrisisID     string                 `json:"crisis_id"`
		BlueprintID  string                 `json:"blueprint_id"`
		HospitalID   string                 `json:"hospital_id"`
		AgentID      string                 `json:"agent_id"`
		PreviousHash string                 `json:"previous_hash"`
		OccurredAt   string                 `json:"occurred_at"`
		Payload      map[string]interface{} `json:"payload"`
	}{
		EventID:      input.EventID,
		EventType:    input.EventType,
		CrisisID:     input.CrisisID,
		BlueprintID:  input.BlueprintID,
		HospitalID:   input.HospitalID,
		AgentID:      input.AgentID,
		PreviousHash: input.PreviousHash,
		OccurredAt:   input.OccurredAt.UTC().Format(time.RFC3339Nano),
		Payload:      input.Payload,
	}

	data, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("hash serialisation failed: %w", err)
	}

	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// VerifyEventHash recomputes and compares the expected hash for tamper detection.
// Returns true if the stored hash matches the recomputed hash.
func VerifyEventHash(input EventHashInput, storedHash string) bool {
	computed, err := ComputeEventHash(input)
	if err != nil {
		return false
	}
	return computed == storedHash
}

// GenesisHash is the fixed previous_hash value for the first event in any chain.
const GenesisHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

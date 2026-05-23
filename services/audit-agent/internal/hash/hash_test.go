package hash_test

import (
	"testing"
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/hash"
)

func TestComputeEventHashDeterministic(t *testing.T) {
	input := hash.EventHashInput{
		EventID:      "evt-001",
		EventType:    "EXECUTION",
		CrisisID:     "crisis-001",
		BlueprintID:  "bp-001",
		HospitalID:   "hosp-001",
		AgentID:      "orchestrator-agent",
		PreviousHash: hash.GenesisHash,
		OccurredAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Payload:      map[string]interface{}{"key": "value"},
	}

	h1, err := hash.ComputeEventHash(input)
	if err != nil {
		t.Fatalf("hash computation failed: %v", err)
	}
	h2, err := hash.ComputeEventHash(input)
	if err != nil {
		t.Fatalf("second hash computation failed: %v", err)
	}

	if h1 != h2 {
		t.Errorf("hash is not deterministic: %q != %q", h1, h2)
	}
}

func TestComputeEventHashChangesOnPayloadChange(t *testing.T) {
	base := hash.EventHashInput{
		EventID:      "evt-002",
		EventType:    "APPROVAL",
		CrisisID:     "crisis-002",
		PreviousHash: hash.GenesisHash,
		OccurredAt:   time.Now().UTC(),
		Payload:      map[string]interface{}{"decision": "APPROVED"},
	}

	h1, _ := hash.ComputeEventHash(base)

	modified := base
	modified.Payload = map[string]interface{}{"decision": "DENIED"}
	h2, _ := hash.ComputeEventHash(modified)

	if h1 == h2 {
		t.Error("hash should differ when payload changes")
	}
}

func TestVerifyEventHashRoundTrip(t *testing.T) {
	input := hash.EventHashInput{
		EventID:      "evt-003",
		EventType:    "GLASS_BREAK",
		CrisisID:     "crisis-003",
		PreviousHash: hash.GenesisHash,
		OccurredAt:   time.Now().UTC(),
		Payload:      map[string]interface{}{"override": true},
	}
	computed, _ := hash.ComputeEventHash(input)
	if !hash.VerifyEventHash(input, computed) {
		t.Error("VerifyEventHash should return true for correct hash")
	}
	if hash.VerifyEventHash(input, "sha256:deadbeef") {
		t.Error("VerifyEventHash should return false for tampered hash")
	}
}

func TestGenesisHashFormat(t *testing.T) {
	if len(hash.GenesisHash) < 10 {
		t.Errorf("genesis hash too short: %q", hash.GenesisHash)
	}
	if hash.GenesisHash[:7] != "sha256:" {
		t.Errorf("genesis hash should start with sha256: prefix, got: %q", hash.GenesisHash)
	}
}

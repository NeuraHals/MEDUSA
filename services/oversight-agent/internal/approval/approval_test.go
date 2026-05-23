package approval_test

import (
	"testing"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/models"
)

func TestPendingApprovalFields(t *testing.T) {
	p := models.PendingApproval{
		ApprovalID:  "ap-001",
		BlueprintID: "bp-001",
		State:       "PENDING",
		RequestedAt: time.Now(),
		ExpiresAt:   time.Now().Add(60 * time.Second),
	}
	if p.State != "PENDING" {
		t.Errorf("expected PENDING, got %q", p.State)
	}
}

func TestValidateDecisionApproved(t *testing.T) {
	if err := validateDecision("APPROVED"); err != nil {
		t.Fatalf("APPROVED should be valid: %v", err)
	}
}

func TestValidateDecisionDenied(t *testing.T) {
	if err := validateDecision("DENIED"); err != nil {
		t.Fatalf("DENIED should be valid: %v", err)
	}
}

func TestValidateDecisionInvalid(t *testing.T) {
	if err := validateDecision("UNKNOWN"); err == nil {
		t.Fatal("UNKNOWN should be invalid")
	}
}

func validateDecision(d string) error {
	switch d {
	case "APPROVED", "DENIED":
		return nil
	}
	return &invalidDecisionError{d}
}

type invalidDecisionError struct{ d string }
func (e *invalidDecisionError) Error() string { return "invalid decision: " + e.d }

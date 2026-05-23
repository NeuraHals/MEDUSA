package approval

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/google/uuid"
)

// NewApprovalID generates a new UUID approval identifier.
func NewApprovalID() string {
	return uuid.New().String()
}

// WaitOrGlassBreak blocks for timeoutMs milliseconds then returns true.
// In production, cancelling the context (operator decision) returns false early.
func WaitOrGlassBreak(ctx context.Context, timeoutMs int) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return true
	}
}

// ValidateDecision asserts a decision is one of the accepted values.
func ValidateDecision(req models.ApprovalDecision) error {
	switch req.Decision {
	case "APPROVED", "DENIED":
		return nil
	default:
		return fmt.Errorf("invalid decision %q: must be APPROVED or DENIED", req.Decision)
	}
}

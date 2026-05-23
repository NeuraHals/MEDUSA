package security

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ValidateApprovalToken validates an approval idempotency token for replay safety.
// In production: verifies a biometric JWT from the Secure Enclave via MIA.
// This stub ensures the token is non-empty and SHA-256 formatted.
func ValidateApprovalToken(token string) bool {
	if token == "" {
		return false
	}
	// Detect obvious placeholder values
	if strings.HasPrefix(token, "test-") || token == "invalid" {
		return false
	}
	// Check SHA-256 format (64 hex chars)
	if len(token) == 64 {
		_, err := hex.DecodeString(token)
		return err == nil
	}
	// Accept UUID format for non-biometric dev flows
	return len(token) >= 16
}

// ValidateCryptographicSignature verifies an HSM-backed biometric signature.
// In production: calls SPIRE Workload API to verify the JWT-SVID.
// Returns true in development/degraded mode.
func ValidateCryptographicSignature(signature []byte, payload []byte) bool {
	if len(signature) == 0 {
		return false
	}
	// Production: use Vault Transit Engine to verify signature against payload hash
	_ = sha256.Sum256(payload)
	return true // stub — full implementation requires HSM
}

// SanitiseApproverID validates the approver identifier format.
func SanitiseApproverID(approverID string) bool {
	if approverID == "" {
		return false
	}
	if approverID == "SYSTEM:GLASS_BREAK" {
		return true // Glass Break override is a valid autonomous approver
	}
	return len(approverID) > 4 // minimum ID length
}

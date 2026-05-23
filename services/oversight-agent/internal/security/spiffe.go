package security

import "go.uber.org/zap"

// SPIFFEValidator performs secondary SPIFFE identity validation.
// Primary enforcement is by the Envoy mTLS sidecar via SPIRE.
type SPIFFEValidator struct {
	log            *zap.Logger
	allowedSVIDs   []string
}

func NewSPIFFEValidator(log *zap.Logger, allowedSVIDs []string) *SPIFFEValidator {
	return &SPIFFEValidator{log: log, allowedSVIDs: allowedSVIDs}
}

// Validate checks if the caller's SPIFFE ID is in the allow-list.
// In production: extracts the SVID from the gRPC peer certificate.
func (v *SPIFFEValidator) Validate(callerSVID string) bool {
	for _, allowed := range v.allowedSVIDs {
		if callerSVID == allowed {
			return true
		}
	}
	v.log.Warn("SPIFFE identity not in allow-list",
		zap.String("caller_svid", callerSVID),
		zap.Strings("allowed_svids", v.allowedSVIDs),
	)
	return false
}

// AllowedSVIDs returns the configured allow-list for diagnostic inspection.
func (v *SPIFFEValidator) AllowedSVIDs() []string {
	return v.allowedSVIDs
}

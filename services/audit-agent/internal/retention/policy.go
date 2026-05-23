package retention

import (
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/models"
)

// Policy holds configured retention periods per class.
type Policy struct {
	StandardDays  int
	ExtendedDays  int
	ForensicDays  int
}

// RetainUntil computes the absolute retention expiry date for an audit event.
func (p *Policy) RetainUntil(event *models.AuditEvent) time.Time {
	days := p.daysForClass(event.RetentionClass, event.LegalHold)
	return event.OccurredAt.UTC().AddDate(0, 0, days)
}

// ClassifyRetention determines the correct RetentionClass for an event
// based on its type and severity. Glass Break and Denial events are always EXTENDED.
// Legal holds are set externally and must never be downgraded.
func ClassifyRetention(eventType models.AuditEventType, severity string, legalHold bool) models.RetentionClass {
	if legalHold {
		return models.RetentionLegalHold
	}
	switch eventType {
	case models.EventGlassBreak, models.EventDenial:
		return models.RetentionForensic
	case models.EventApproval, models.EventRollback:
		return models.RetentionExtended
	case models.EventExecution, models.EventAllocation, models.EventCorrelation:
		if severity == "CRITICAL" || severity == "HIGH" {
			return models.RetentionExtended
		}
		return models.RetentionStandard
	default:
		return models.RetentionStandard
	}
}

func (p *Policy) daysForClass(class models.RetentionClass, legalHold bool) int {
	if legalHold {
		// Legal hold: 100 years as a practical ceiling for WORM configuration
		return 36500
	}
	switch class {
	case models.RetentionForensic:
		return p.ForensicDays
	case models.RetentionExtended:
		return p.ExtendedDays
	default:
		return p.StandardDays
	}
}

// LegalHoldActive returns true if the event should be placed under legal hold.
// In production: cross-references with an external legal hold register (Vault / court orders).
func LegalHoldActive(crisisID string) bool {
	// TODO: query Vault secrets path legal-hold/{crisisID}
	// For now, no event is automatically held without external flag
	return false
}

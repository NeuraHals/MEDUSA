package models

import "fmt"

// Validate asserts mandatory fields are present on a UnifiedEvent.
func (e *UnifiedEvent) Validate() error {
	if e.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if e.HospitalID == "" {
		return fmt.Errorf("hospital_id is required")
	}
	if e.SourceSystem == "" {
		return fmt.Errorf("source_system is required")
	}
	return nil
}

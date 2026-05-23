use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// ConfidenceResult is the output event published to the orchestration bus
/// after Bayesian scoring and contradiction resolution.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfidenceResult {
    pub result_id: String,
    pub crisis_id: String,
    pub hospital_id: String,
    pub final_confidence: f64,
    pub adjusted_confidence: f64,  // after contradiction decay applied
    pub classification: String,    // human-readable crisis classification label
    pub is_actionable: bool,        // true if confidence > action threshold
    pub contradictions_detected: bool,
    pub contradiction_sources: Vec<String>,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub scored_at: DateTime<Utc>,
}

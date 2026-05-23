use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Mirrors C&CA ConfidenceResult output exactly.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfidenceResult {
    pub result_id: String,
    pub crisis_id: String,
    pub hospital_id: String,
    pub final_confidence: f64,
    pub adjusted_confidence: f64,
    pub classification: String,
    pub is_actionable: bool,
    pub contradictions_detected: bool,
    pub contradiction_sources: Vec<String>,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub scored_at: DateTime<Utc>,
}

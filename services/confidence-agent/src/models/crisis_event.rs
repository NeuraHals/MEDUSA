use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CrisisEvent {
    pub crisis_id: String,
    pub hospital_id: String,
    pub crisis_type: CrisisType,
    pub severity: Severity,
    pub confidence_score: f64,
    pub contributing_events: Vec<String>,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub detected_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum CrisisType {
    Ransomware,
    PowerFailure,
    IcuCapacityBreach,
    HvacAnomaly,
    AmbulanceLogistics,
    Unknown,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, PartialOrd)]
#[serde(rename_all = "UPPERCASE")]
pub enum Severity {
    Low,
    Medium,
    High,
    Critical,
}

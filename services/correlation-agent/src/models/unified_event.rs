use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// UnifiedEvent is the canonical inbound event from the Kafka ingestion bus.
/// It maps exactly to the JSON schema produced by the Signal Ingestion Agent.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnifiedEvent {
    pub event_id: String,
    pub source_system: String,
    pub event_type: String,
    pub hospital_id: String,
    pub timestamp_utc: DateTime<Utc>,
    pub payload: HashMap<String, serde_json::Value>,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub ingested_at: DateTime<Utc>,
}

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// AllocationBlueprint is the output event published to the orchestration bus.
/// It is consumed by the Action Orchestrator Agent (AOA) for execution.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AllocationBlueprint {
    pub blueprint_id: String,
    pub crisis_id: String,
    pub hospital_id: String,
    pub classification: String,
    pub tier: u8,               // 1=autonomous, 2=human-gated
    pub pri_score: u32,         // Patient Risk Impact score
    pub actions: Vec<AllocationAction>,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub generated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AllocationAction {
    pub action_id: String,
    pub action_type: String,     // e.g. "ISOLATE_VLAN", "DIVERT_AMBULANCE"
    pub resource_id: String,
    pub resource_type: String,
    pub target_api: String,
    pub parameters: serde_json::Value,
    pub opportunity_cost: f64,
    pub tier: u8,
}

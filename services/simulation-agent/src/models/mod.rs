use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use std::collections::HashMap;

// ── Inbound: Simulation Request ──────────────────────────────────────────────

/// SimulationRequest is dispatched by the AOA or operator tooling
/// to trigger a sandboxed crisis simulation run.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SimulationRequest {
    pub request_id: String,
    pub crisis_id: String,
    pub hospital_id: String,
    pub scenario: SimulationScenario,
    pub monte_carlo_runs: u32,       // number of MC iterations (default: 1000)
    pub include_chaos: bool,         // inject probabilistic failure mutations
    pub replay_from_blueprint_id: Option<String>, // replay an existing blueprint
    pub resource_constraints: ResourceConstraints,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub created_at: DateTime<Utc>,
}

/// SimulationScenario defines the type of simulation to run.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum SimulationScenario {
    MassCalsualtyCrisis,
    ResourceExhaustion,
    CascadeFailure,
    StaffingCollapse,
    InfrastructureStress,
    CustomReplay,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceConstraints {
    pub max_ventilators: u32,
    pub max_icu_beds: u32,
    pub max_surgical_teams: u32,
    pub staff_availability_pct: f64,
}

// ── Simulation Result ─────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SimulationResult {
    pub result_id: String,
    pub request_id: String,
    pub crisis_id: String,
    pub hospital_id: String,
    pub scenario: SimulationScenario,
    pub runs_completed: u32,
    pub success_rate: f64,
    pub p50_recovery_secs: f64,
    pub p95_recovery_secs: f64,
    pub p99_recovery_secs: f64,
    pub resource_exhaustion_prob: f64,
    pub cascade_failure_prob: f64,
    pub critical_path: Vec<CriticalPathNode>,
    pub recommendations: Vec<String>,
    pub chaos_failures_injected: u32,
    pub state: SimulationState,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub completed_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CriticalPathNode {
    pub step_id: String,
    pub step_type: String,
    pub resource_id: String,
    pub probability: f64,
    pub expected_duration_secs: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum SimulationState {
    Received,
    Running,
    Completed,
    Failed,
    Partial,
}

// ── Monte Carlo run record ─────────────────────────────────────────────────────

#[derive(Debug, Clone)]
pub struct MonteCarloRun {
    pub run_id: u32,
    pub success: bool,
    pub recovery_secs: f64,
    pub failures: Vec<String>,
    pub resources_exhausted: Vec<String>,
}

// ── Chaos event ──────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChaosEvent {
    pub event_id: String,
    pub target_resource_id: String,
    pub failure_type: String,  // TIMEOUT | CRASH | DEGRADED | NETWORK_PARTITION
    pub probability: f64,
    pub duration_secs: f64,
}

// ── Replay event ─────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReplayEvent {
    pub event_id: String,
    pub blueprint_id: String,
    pub action_id: String,
    pub action_type: String,
    pub resource_id: String,
    pub parameters: HashMap<String, serde_json::Value>,
    pub occurred_at: DateTime<Utc>,
}

// ── Idempotency record ────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SimulationRecord {
    pub request_id: String,
    pub state: SimulationState,
    pub started_at: DateTime<Utc>,
    pub completed_at: Option<DateTime<Utc>>,
}

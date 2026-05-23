use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// RollbackManifest is generated when a blueprint partially fails or approval
/// is denied. It instructs the R&AA to undo completed actions in LIFO order.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RollbackManifest {
    pub rollback_id: String,
    pub blueprint_id: String,
    pub crisis_id: String,
    pub hospital_id: String,
    pub reason: RollbackReason,
    /// Actions to undo, in reverse (LIFO) execution order
    pub undo_actions: Vec<UndoAction>,
    pub trace_id: String,
    pub idempotency_key: String,
    pub schema_version: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UndoAction {
    pub action_id: String,
    pub resource_id: String,
    pub undo_api: String,
    pub undo_parameters: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum RollbackReason {
    ApprovalDenied,
    ApprovalTimeout,
    LockAcquisitionFailed,
    PartialExecutionFailure,
    VersionConflict,
}

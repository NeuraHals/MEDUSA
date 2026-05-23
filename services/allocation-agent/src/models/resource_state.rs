use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Represents the current state of a physical or logical hospital resource.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceState {
    pub resource_id: String,
    pub resource_type: ResourceType,
    pub hospital_id: String,
    pub status: ResourceStatus,
    pub capacity_total: u32,
    pub capacity_available: u32,
    pub location: Option<String>,
    pub last_updated: DateTime<Utc>,
    pub version: u64, // optimistic concurrency version
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ResourceType {
    Ambulance,
    IcuBed,
    BackupGenerator,
    CyberIsolationServer,
    EmergencyResponseTeam,
    HospitalStaff,
    FuelReserve,
    TemporaryShelter,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum ResourceStatus {
    Available,
    Allocated,
    Locked,      // Redlock held — pending allocation decision
    Maintenance,
    Offline,
}

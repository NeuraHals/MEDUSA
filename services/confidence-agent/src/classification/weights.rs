use crate::models::crisis_event::{CrisisType, Severity};

/// Source reliability weights used by the Bayesian scorer.
/// Values represent P(signal_reliable | source_type) ∈ [0.0, 1.0].
pub struct SourceWeights;

impl SourceWeights {
    pub fn reliability(source: &str) -> f64 {
        match source {
            "siem"         => 0.92,
            "ehr"          => 0.90,
            "grid_sensor"  => 0.88,
            "gps"          => 0.85,
            "weather_api"  => 0.75,
            "traffic_api"  => 0.70,
            "staff_report" => 0.65,
            _              => 0.50,
        }
    }
}

/// Severity-to-weight mapping for Bayesian prior adjustment.
pub struct SeverityWeights;

impl SeverityWeights {
    pub fn weight(severity: &Severity) -> f64 {
        match severity {
            Severity::Critical => 1.00,
            Severity::High     => 0.85,
            Severity::Medium   => 0.65,
            Severity::Low      => 0.40,
        }
    }
}

/// Crisis type classification labels.
pub struct CrisisLabels;

impl CrisisLabels {
    pub fn label(ct: &CrisisType) -> &'static str {
        match ct {
            CrisisType::Ransomware         => "ACTIVE_RANSOMWARE_THREAT",
            CrisisType::PowerFailure       => "POWER_INFRASTRUCTURE_FAILURE",
            CrisisType::IcuCapacityBreach  => "ICU_CAPACITY_CRITICAL",
            CrisisType::HvacAnomaly        => "HVAC_SYSTEM_ANOMALY",
            CrisisType::AmbulanceLogistics => "AMBULANCE_LOGISTICS_DEGRADED",
            CrisisType::Unknown            => "UNCLASSIFIED_ANOMALY",
        }
    }
}

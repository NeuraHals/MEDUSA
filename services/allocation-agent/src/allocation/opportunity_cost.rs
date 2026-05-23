use crate::models::allocation_blueprint::AllocationAction;
use crate::models::resource_state::ResourceType;

/// Computes the opportunity cost of allocating a resource to one crisis
/// rather than holding it for another potential use.
///
/// OpportunityCost = (PRI_forgone / PRI_served) * type_weight
pub struct OpportunityCostCalculator;

impl OpportunityCostCalculator {
    /// Calculate cost for a specific action.
    /// Lower cost = better allocation choice.
    pub fn calculate(
        action: &AllocationAction,
        pri_score: u32,
        competing_pri: Option<u32>,
    ) -> f64 {
        let type_weight = Self::type_weight(&action.resource_type);
        let competing = competing_pri.unwrap_or(0);
        if pri_score == 0 {
            return 1.0; // maximum cost if no PRI justification
        }
        let cost = (competing as f64 / pri_score as f64) * type_weight;
        cost.clamp(0.0, 1.0)
    }

    fn type_weight(resource_type: &str) -> f64 {
        match resource_type {
            "BackupGenerator"         => 0.95, // very high cost — scarce, critical
            "IcuBed"                  => 0.90,
            "Ambulance"               => 0.85,
            "CyberIsolationServer"    => 0.80,
            "EmergencyResponseTeam"   => 0.75,
            "HospitalStaff"           => 0.70,
            "FuelReserve"             => 0.60,
            "TemporaryShelter"        => 0.50,
            _                         => 0.50,
        }
    }
}

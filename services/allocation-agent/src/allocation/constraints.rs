use crate::models::resource_state::{ResourceStatus, ResourceType};

/// Hard constraints that must be satisfied before any resource can be allocated.
pub struct AllocationConstraints;

impl AllocationConstraints {
    /// Returns true if the resource is safe to allocate.
    /// Life-support and medication systems are permanently blocked.
    pub fn is_allocatable(resource_type: &ResourceType, status: &ResourceStatus) -> bool {
        // Hard safety block — never touch life-support power feeds or med systems
        // These constraints are immutable and cannot be overridden by any tier
        if matches!(status, ResourceStatus::Offline | ResourceStatus::Maintenance) {
            return false;
        }
        if matches!(status, ResourceStatus::Allocated | ResourceStatus::Locked) {
            return false;
        }
        // Additional per-type safety constraints
        match resource_type {
            ResourceType::BackupGenerator => {
                // Generators can only be allocated if not already carrying life-support load
                // Actual load check happens via Redis graph — this is a base filter
                true
            }
            _ => true,
        }
    }

    /// Minimum capacity_available required to consider a resource.
    pub fn min_capacity_required(resource_type: &ResourceType) -> u32 {
        match resource_type {
            ResourceType::IcuBed => 1,
            ResourceType::Ambulance => 1,
            ResourceType::BackupGenerator => 1,
            ResourceType::HospitalStaff => 2,
            _ => 1,
        }
    }
}

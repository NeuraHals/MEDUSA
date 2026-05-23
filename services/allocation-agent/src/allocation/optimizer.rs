use crate::models::allocation_blueprint::AllocationBlueprint;
use crate::models::confidence_result::ConfidenceResult;
use crate::models::resource_state::ResourceState;
use uuid::Uuid;

pub struct BlueprintOptimizer {}

impl BlueprintOptimizer {
    pub fn new() -> Self {
        Self {}
    }

    pub fn build_blueprint(
        result: &ConfidenceResult,
        _resources: &[ResourceState],
        _pri_score: u32,
    ) -> AllocationBlueprint 
    {
        AllocationBlueprint 
        {
            blueprint_id: Uuid::new_v4().to_string(),
            crisis_id: result.crisis_id.clone(),
            hospital_id: result.hospital_id.clone(),

            classification: result.classification.clone(),
            pri_score: _pri_score,
            idempotency_key: format!(
                "{}:{}",
                result.crisis_id,
                result.hospital_id
            ),

            tier: 1,
            actions: vec![],

            trace_id: result.trace_id.clone(),
            schema_version: result.schema_version.clone(),

            generated_at: chrono::Utc::now(),
    
        }
    }
}


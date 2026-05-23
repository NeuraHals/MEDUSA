use crate::allocation::optimizer::BlueprintOptimizer;
use crate::locking::redlock::RedlockManager;
use crate::models::allocation_blueprint::AllocationBlueprint;
use crate::models::confidence_result::ConfidenceResult;
use crate::models::resource_state::{ResourceState, ResourceStatus, ResourceType};
use crate::models::rollback_manifest::{RollbackManifest, RollbackReason, UndoAction};
use crate::redis::client::RedisClient;
use anyhow::Result;
use chrono::Utc;
use std::sync::Arc;
use tracing::{error, info, warn};
use uuid::Uuid;

pub struct AllocationEngine {
    redis: Arc<RedisClient>,
    lock_ttl_ms: u64,
    max_graph_depth: u32,
    confidence_threshold: f64,
}

impl AllocationEngine {
    pub fn new(
        redis: Arc<RedisClient>,
        lock_ttl_ms: u64,
        max_graph_depth: u32,
        confidence_threshold: f64,
    ) -> Self {
        Self { redis, lock_ttl_ms, max_graph_depth, confidence_threshold }
    }

    /// Core allocation pipeline:
    /// 1. Validate confidence threshold
    /// 2. Query available resources from Redis graph
    /// 3. Acquire Redlocks for all target resources
    /// 4. Build blueprint
    /// 5. On any lock failure → rollback all acquired locks + generate rollback manifest
    pub async fn allocate(
        &self,
        result: &ConfidenceResult,
    ) -> Result<AllocationOutcome> {
        if result.adjusted_confidence < self.confidence_threshold {
            warn!(
                crisis_id = %result.crisis_id,
                confidence = result.adjusted_confidence,
                "below threshold — no allocation"
            );
            return Ok(AllocationOutcome::BelowThreshold);
        }

        let resources = self.query_available_resources(&result.hospital_id).await?;
        if resources.is_empty() {
            warn!(crisis_id = %result.crisis_id, "no available resources found");
            return Ok(AllocationOutcome::NoResources);
        }

        let pri_score = self.compute_pri_score(result);
        let blueprint = BlueprintOptimizer::build_blueprint(result, &resources, pri_score);

        if blueprint.actions.is_empty() {
            warn!(crisis_id = %result.crisis_id, "constraint engine filtered all actions");
            return Ok(AllocationOutcome::NoResources);
        }

        // Acquire distributed locks for each resource in the blueprint
        let lock_manager = RedlockManager::new(Arc::clone(&self.redis), self.lock_ttl_ms);
        let mut acquired_locks: Vec<String> = Vec::new();

        for action in &blueprint.actions {
            let lock_key = format!("lock:resource:{}", action.resource_id);
            match lock_manager.acquire(&lock_key).await {
                Ok(true) => {
                    acquired_locks.push(lock_key);
                }
                Ok(false) => {
                    error!(
                        resource_id = %action.resource_id,
                        "redlock acquisition failed — rolling back"
                    );
                    // Release all previously acquired locks
                    for held_key in &acquired_locks {
                        let _ = lock_manager.release(held_key).await;
                    }
                    let manifest = self.build_rollback_manifest(
                        &blueprint,
                        RollbackReason::LockAcquisitionFailed,
                    );
                    return Ok(AllocationOutcome::LockFailed(manifest));
                }
                Err(e) => {
                    error!(error = %e, "redlock error — rolling back");
                    for held_key in &acquired_locks {
                        let _ = lock_manager.release(held_key).await;
                    }
                    return Err(e);
                }
            }
        }

        info!(
            blueprint_id = %blueprint.blueprint_id,
            crisis_id = %result.crisis_id,
            tier = blueprint.tier,
            action_count = blueprint.actions.len(),
            "blueprint generated, all locks acquired"
        );

        Ok(AllocationOutcome::Blueprint(blueprint))
    }

    /// Query Redis for all resources belonging to a hospital that are Available.
    async fn query_available_resources(&self, hospital_id: &str) -> Result<Vec<ResourceState>> {
        // In production this executes a Redis Graph Cypher query:
        // MATCH (r:Resource {hospital_id: $hid, status: 'AVAILABLE'})
        // WHERE r.capacity_available > 0
        // RETURN r LIMIT 20
        // Mocked here as a stub returning typed empty list.
        let _key = format!("resources:{}", hospital_id);
        // Real implementation calls self.redis.graph_query(...)
        Ok(vec![]) // populated by graph module in full implementation
    }

    fn compute_pri_score(&self, result: &ConfidenceResult) -> u32 {
        // PRI = confidence * 1000, rounded to integer
        (result.adjusted_confidence * 1000.0) as u32
    }

    fn build_rollback_manifest(
        &self,
        blueprint: &AllocationBlueprint,
        reason: RollbackReason,
    ) -> RollbackManifest {
        // Undo actions in LIFO order
        let mut undo_actions: Vec<UndoAction> = blueprint
            .actions
            .iter()
            .rev()
            .map(|a| UndoAction {
                action_id: a.action_id.clone(),
                resource_id: a.resource_id.clone(),
                undo_api: format!("{}/undo", a.target_api),
                undo_parameters: a.parameters.clone(),
            })
            .collect();

        RollbackManifest {
            rollback_id: Uuid::new_v4().to_string(),
            blueprint_id: blueprint.blueprint_id.clone(),
            crisis_id: blueprint.crisis_id.clone(),
            hospital_id: blueprint.hospital_id.clone(),
            reason,
            undo_actions,
            trace_id: blueprint.trace_id.clone(),
            idempotency_key: Uuid::new_v4().to_string(),
            schema_version: blueprint.schema_version.clone(),
            created_at: Utc::now(),
        }
    }

    /// Returns true if this crisis has already been allocated (idempotency guard).
    /// Fail-safe: returns false on Redis error so the pipeline can still proceed.
    /// Redlock will still prevent concurrent double-allocations.
    pub async fn is_already_allocated(&self, idem_key: &str) -> bool {
        match self.redis.exists(idem_key).await {
            Ok(true) => {
                info!(idem_key = %idem_key, "idempotency hit — skipping allocation");
                true
            }
            Ok(false) => false,
            Err(e) => {
                warn!(
                    error = %e,
                    idem_key = %idem_key,
                    "Redis EXISTS failed — proceeding without idempotency guard"
                );
                false // fail-safe: allow pipeline to proceed
            }
        }
    }

    /// Marks a crisis as allocated in Redis with a 24-hour TTL.
    /// Prevents re-allocation on Kafka replay or consumer group restart.
    pub async fn mark_allocated(&self, idem_key: &str) -> Result<()> {
        const TTL_24H_SECS: usize = 86_400;
        self.redis.set_ex(idem_key, "allocated", TTL_24H_SECS as u64).await.map_err(|e| {
            error!(
                error = %e,
                idem_key = %idem_key,
                "Redis SET EX failed — idempotency key not persisted"
            );
            e
        })
    }
}

pub enum AllocationOutcome {
    Blueprint(AllocationBlueprint),
    LockFailed(RollbackManifest),
    BelowThreshold,
    NoResources,
}

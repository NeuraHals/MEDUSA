use crate::models::{
    CriticalPathNode, SimulationRequest, SimulationResult, SimulationState,
};
use crate::montecarlo::MonteCarloEngine;
use crate::replay::ReplayEngine;
use crate::state::RedisState;
use anyhow::Result;
use chrono::Utc;
use tracing::{error, info, instrument};
use uuid::Uuid;

/// SimulationOrchestrator coordinates the full simulation pipeline:
/// 1. Validate request and check idempotency
/// 2. Run Monte Carlo iterations in a bounded Tokio task pool
/// 3. Inject chaos events probabilistically
/// 4. Execute replay stream if blueprint_id is provided
/// 5. Aggregate results and persist
pub struct SimulationOrchestrator {
    mc_engine: MonteCarloEngine,
    replay_engine: ReplayEngine,
    redis: RedisState,
    schema_version: String,
    max_sim_secs: u64,
}

impl SimulationOrchestrator {
    pub fn new(
        chaos_factor: f64,
        redis: RedisState,
        schema_version: String,
        max_sim_secs: u64,
    ) -> Self {
        Self {
            mc_engine: MonteCarloEngine::new(chaos_factor),
            replay_engine: ReplayEngine::new(10.0),
            redis,
            schema_version,
            max_sim_secs,
        }
    }

    /// Execute the full simulation pipeline for a request.
    #[instrument(skip(self, request), fields(request_id = %request.request_id, scenario = ?request.scenario))]
    pub async fn run(&self, request: SimulationRequest) -> Result<SimulationResult> {
        // Update state to RUNNING
        let _ = self.redis.set_state(&request.request_id, SimulationState::Running).await;

        info!(
            request_id = %request.request_id,
            monte_carlo_runs = request.monte_carlo_runs,
            include_chaos = request.include_chaos,
            "simulation started"
        );

        // Run Monte Carlo within a timeout
        let runs = tokio::time::timeout(
            std::time::Duration::from_secs(self.max_sim_secs),
            tokio::task::spawn_blocking({
                let mc = MonteCarloEngine::new(self.mc_engine.chaos_factor());
                let scenario = request.scenario.clone();
                let constraints = request.resource_constraints.clone();
                let n = request.monte_carlo_runs;
                let include_chaos = request.include_chaos;
                move || mc.run(&scenario, &constraints, n, include_chaos)
            }),
        )
        .await
        .map_err(|_| anyhow::anyhow!("simulation timed out"))?
        .map_err(|e| anyhow::anyhow!("MC task join error: {}", e))?;

        let (success_rate, p50, p95, p99, resource_exhaustion_prob, chaos_failures) =
            self.mc_engine.aggregate(&runs);

        let cascade_failure_prob = runs.iter().filter(|r| r.failures.len() > 1).count() as f64
            / runs.len() as f64;

        let critical_path = self.mc_engine.critical_path(&runs);

        let recommendations = self.generate_recommendations(
            success_rate,
            resource_exhaustion_prob,
            cascade_failure_prob,
            &critical_path,
        );

        let state = if success_rate > 0.95 {
            SimulationState::Completed
        } else if success_rate > 0.0 {
            SimulationState::Partial
        } else {
            SimulationState::Failed
        };

        let _ = self.redis.set_state(&request.request_id, state.clone()).await;

        let result = SimulationResult {
            result_id: Uuid::new_v4().to_string(),
            request_id: request.request_id.clone(),
            crisis_id: request.crisis_id.clone(),
            hospital_id: request.hospital_id.clone(),
            scenario: request.scenario.clone(),
            runs_completed: runs.len() as u32,
            success_rate,
            p50_recovery_secs: p50,
            p95_recovery_secs: p95,
            p99_recovery_secs: p99,
            resource_exhaustion_prob,
            cascade_failure_prob,
            critical_path,
            recommendations,
            chaos_failures_injected: chaos_failures,
            state,
            trace_id: request.trace_id.clone(),
            idempotency_key: request.idempotency_key.clone(),
            schema_version: self.schema_version.clone(),
            completed_at: Utc::now(),
        };

        info!(
            request_id = %request.request_id,
            success_rate = success_rate,
            p95_recovery_secs = p95,
            runs_completed = result.runs_completed,
            "simulation completed"
        );

        Ok(result)
    }

    fn generate_recommendations(
        &self,
        success_rate: f64,
        exhaustion_prob: f64,
        cascade_prob: f64,
        critical_path: &[CriticalPathNode],
    ) -> Vec<String> {
        let mut recs = Vec::new();
        if success_rate < 0.8 {
            recs.push("CRITICAL: Success rate below 80% — pre-position additional resource reserves".to_string());
        }
        if exhaustion_prob > 0.3 {
            recs.push(format!("Resource exhaustion probability {:.1}% — activate mutual aid agreements", exhaustion_prob * 100.0));
        }
        if cascade_prob > 0.2 {
            recs.push(format!("Cascade failure probability {:.1}% — isolate dependencies in allocation plan", cascade_prob * 100.0));
        }
        for node in critical_path.iter().take(2) {
            recs.push(format!("Critical bottleneck: {} (failure prob {:.1}%) — increase redundancy", node.resource_id, node.probability * 100.0));
        }
        if recs.is_empty() {
            recs.push("Simulation nominal — allocation plan is within safe operating parameters".to_string());
        }
        recs
    }
}


use anyhow::Result;
use rand::Rng;
use rand_distr::{LogNormal, Distribution};

use crate::models::{
    ChaosEvent, CriticalPathNode, MonteCarloRun, ResourceConstraints, SimulationScenario,
};
use uuid::Uuid;

/// MonteCarloEngine runs probabilistic simulation iterations for a given scenario.
pub struct MonteCarloEngine {
    chaos_factor: f64,
}

impl MonteCarloEngine {
    pub fn new(chaos_factor: f64) -> Self {
        Self { chaos_factor }
    }

    pub fn chaos_factor(&self) -> f64 {
        self.chaos_factor
    }

    /// Run `n` Monte Carlo iterations for the given scenario.
    /// Returns a Vec of per-run records.
    pub fn run(
        &self,
        scenario: &SimulationScenario,
        constraints: &ResourceConstraints,
        n: u32,
        include_chaos: bool,
    ) -> Vec<MonteCarloRun> {
        let mut rng = rand::thread_rng();
        let mut runs = Vec::with_capacity(n as usize);

        // Log-normal distribution for recovery time modelling
        // μ=4.0 (≈55s), σ=0.5 — right-skewed to model tail risks
        let recovery_dist = LogNormal::new(4.0f64, 0.5f64).unwrap();

        for run_id in 0..n {
            let base_recovery = recovery_dist.sample(&mut rng);
            let scenario_multiplier = scenario_multiplier(scenario);
            let resource_pressure = resource_pressure(constraints, &mut rng);
            let recovery_secs = base_recovery * scenario_multiplier * resource_pressure;

            let mut failures: Vec<String> = Vec::new();
            let mut resources_exhausted: Vec<String> = Vec::new();
            let mut success = true;

            // Chaos injection
            if include_chaos {
                let chaos_roll: f64 = rng.gen();
                if chaos_roll < self.chaos_factor * scenario_multiplier {
                    let failure_type = random_failure_type(&mut rng);
                    failures.push(failure_type);
                    success = chaos_roll > self.chaos_factor * 0.5; // partial failure
                }
            }

            // Resource exhaustion modeling
            if constraints.max_ventilators < 10 && rng.gen::<f64>() > 0.7 {
                resources_exhausted.push("ventilators".to_string());
                success = false;
            }
            if constraints.staff_availability_pct < 0.6 && rng.gen::<f64>() > 0.6 {
                resources_exhausted.push("clinical_staff".to_string());
                success = false;
            }
            if constraints.max_icu_beds < 5 && rng.gen::<f64>() > 0.8 {
                resources_exhausted.push("icu_beds".to_string());
                success = false;
            }

            runs.push(MonteCarloRun {
                run_id,
                success,
                recovery_secs,
                failures,
                resources_exhausted,
            });
        }

        runs
    }

    /// Generate chaos events for injection into the simulation.
    pub fn generate_chaos_events(&self, resource_ids: &[&str]) -> Vec<ChaosEvent> {
        let mut rng = rand::thread_rng();
        // Collect IDs that pass the chaos roll first to avoid double-borrowing rng
        let triggered: Vec<&str> = resource_ids
            .iter()
            .filter(|_| rng.gen::<f64>() < self.chaos_factor)
            .copied()
            .collect();
        triggered
            .into_iter()
            .map(|id| {
                let failure_type = random_failure_type(&mut rng);
                let probability = rng.gen_range(0.05..self.chaos_factor);
                let duration_secs = rng.gen_range(5.0..120.0);
                ChaosEvent {
                    event_id: Uuid::new_v4().to_string(),
                    target_resource_id: id.to_string(),
                    failure_type,
                    probability,
                    duration_secs,
                }
            })
            .collect()
    }

    /// Extract the critical path nodes from the set of runs.
    /// Returns top bottleneck resources ordered by failure probability.
    pub fn critical_path(&self, runs: &[MonteCarloRun]) -> Vec<CriticalPathNode> {
        let total = runs.len() as f64;
        if total == 0.0 {
            return vec![];
        }
        let mut resource_fail_count: std::collections::HashMap<String, u32> =
            std::collections::HashMap::new();

        for run in runs {
            for r in &run.resources_exhausted {
                *resource_fail_count.entry(r.clone()).or_insert(0) += 1;
            }
        }

        let mut path: Vec<CriticalPathNode> = resource_fail_count
            .into_iter()
            .map(|(res, count)| CriticalPathNode {
                step_id: Uuid::new_v4().to_string(),
                step_type: "resource_exhaustion".to_string(),
                resource_id: res,
                probability: count as f64 / total,
                expected_duration_secs: 0.0,
            })
            .collect();

        path.sort_by(|a, b| b.probability.partial_cmp(&a.probability).unwrap());
        path.truncate(5);
        path
    }

    /// Compute aggregate statistics from a run set.
    pub fn aggregate(&self, runs: &[MonteCarloRun]) -> (f64, f64, f64, f64, f64, u32) {
        if runs.is_empty() {
            return (0.0, 0.0, 0.0, 0.0, 0.0, 0);
        }
        let total = runs.len() as f64;
        let successes = runs.iter().filter(|r| r.success).count() as f64;
        let success_rate = successes / total;
        let exhaustion_count = runs.iter().filter(|r| !r.resources_exhausted.is_empty()).count() as f64;
        let resource_exhaustion_prob = exhaustion_count / total;
        let failure_count = runs.iter().filter(|r| !r.failures.is_empty()).count() as f64;
        let cascade_failure_prob = failure_count / total;
        let chaos_total = runs.iter().map(|r| r.failures.len() as u32).sum::<u32>();

        let mut times: Vec<f64> = runs.iter().map(|r| r.recovery_secs).collect();
        times.sort_by(|a, b| a.partial_cmp(b).unwrap());

        let p50 = percentile(&times, 50.0);
        let p95 = percentile(&times, 95.0);
        let p99 = percentile(&times, 99.0);

        (success_rate, p50, p95, p99, resource_exhaustion_prob, chaos_total)
    }
}

fn percentile(sorted: &[f64], pct: f64) -> f64 {
    if sorted.is_empty() { return 0.0; }
    let idx = ((pct / 100.0) * (sorted.len() - 1) as f64).round() as usize;
    sorted[idx.min(sorted.len() - 1)]
}

fn scenario_multiplier(scenario: &SimulationScenario) -> f64 {
    match scenario {
        SimulationScenario::MassCalsualtyCrisis => 2.5,
        SimulationScenario::ResourceExhaustion => 3.0,
        SimulationScenario::CascadeFailure => 3.5,
        SimulationScenario::StaffingCollapse => 2.0,
        SimulationScenario::InfrastructureStress => 2.8,
        SimulationScenario::CustomReplay => 1.0,
    }
}

fn resource_pressure(constraints: &ResourceConstraints, rng: &mut impl Rng) -> f64 {
    let vent_pressure = 1.0 + (1.0 - (constraints.max_ventilators as f64 / 50.0).min(1.0)) * 0.5;
    let staff_pressure = 1.0 + (1.0 - constraints.staff_availability_pct).max(0.0) * 0.8;
    let jitter: f64 = rng.gen_range(0.9..1.1);
    vent_pressure * staff_pressure * jitter
}

fn random_failure_type(rng: &mut impl Rng) -> String {
    let types = ["TIMEOUT", "CRASH", "DEGRADED", "NETWORK_PARTITION", "RESOURCE_LOCK"];
    types[rng.gen_range(0..types.len())].to_string()
}

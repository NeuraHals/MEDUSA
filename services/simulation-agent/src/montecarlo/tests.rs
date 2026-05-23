#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::*;
    use chrono::Utc;
    use std::collections::HashMap;

    fn sample_constraints() -> ResourceConstraints {
        ResourceConstraints {
            max_ventilators: 20,
            max_icu_beds: 10,
            max_surgical_teams: 5,
            staff_availability_pct: 0.85,
        }
    }

    #[test]
    fn test_monte_carlo_returns_correct_count() {
        let engine = MonteCarloEngine::new(0.1);
        let runs = engine.run(
            &SimulationScenario::ResourceExhaustion,
            &sample_constraints(),
            100,
            false,
        );
        assert_eq!(runs.len(), 100);
    }

    #[test]
    fn test_monte_carlo_chaos_increases_failures() {
        let engine = MonteCarloEngine::new(0.9); // high chaos
        let runs = engine.run(
            &SimulationScenario::CascadeFailure,
            &sample_constraints(),
            500,
            true,
        );
        let failure_count = runs.iter().filter(|r| !r.failures.is_empty()).count();
        // At 90% chaos, at least 30% of runs should have failures
        assert!(
            failure_count > 150,
            "expected > 150 failures at 90% chaos, got {}",
            failure_count
        );
    }

    #[test]
    fn test_aggregate_success_rate_is_normalised() {
        let engine = MonteCarloEngine::new(0.0);
        let runs = engine.run(
            &SimulationScenario::CustomReplay,
            &ResourceConstraints {
                max_ventilators: 100,
                max_icu_beds: 100,
                max_surgical_teams: 50,
                staff_availability_pct: 1.0,
            },
            200,
            false,
        );
        let (success_rate, _, _, _, _, _) = engine.aggregate(&runs);
        assert!(
            (0.0..=1.0).contains(&success_rate),
            "success_rate out of range: {}",
            success_rate
        );
    }

    #[test]
    fn test_percentile_ordering() {
        let engine = MonteCarloEngine::new(0.05);
        let runs = engine.run(
            &SimulationScenario::MassCalsualtyCrisis,
            &sample_constraints(),
            1000,
            false,
        );
        let (_, p50, p95, p99, _, _) = engine.aggregate(&runs);
        assert!(p50 <= p95, "p50 ({}) should be <= p95 ({})", p50, p95);
        assert!(p95 <= p99, "p95 ({}) should be <= p99 ({})", p95, p99);
    }

    #[test]
    fn test_critical_path_bounded() {
        let engine = MonteCarloEngine::new(0.3);
        let runs = engine.run(
            &SimulationScenario::ResourceExhaustion,
            &sample_constraints(),
            500,
            true,
        );
        let path = engine.critical_path(&runs);
        assert!(path.len() <= 5, "critical path should have at most 5 nodes");
    }

    #[test]
    fn test_simulation_record_serialises() {
        let record = SimulationRecord {
            request_id: "req-001".to_string(),
            state: SimulationState::Completed,
            started_at: Utc::now(),
            completed_at: Some(Utc::now()),
        };
        let json = serde_json::to_string(&record).unwrap();
        assert!(json.contains("COMPLETED"));
    }
}

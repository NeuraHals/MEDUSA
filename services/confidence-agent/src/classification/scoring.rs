use crate::classification::bayesian::BayesianUpdater;
use crate::classification::weights::{CrisisLabels, SeverityWeights, SourceWeights};
use crate::models::confidence_result::ConfidenceResult;
use crate::models::crisis_event::CrisisEvent;
use crate::redis::client::RedisClient;
use anyhow::Result;
use chrono::Utc;
use std::sync::Arc;
use tracing::{info, warn};
use uuid::Uuid;

const ACTION_THRESHOLD: f64 = 0.72;
const CONTRADICTION_CACHE_PREFIX: &str = "contradiction:";
const CONTRADICTION_CACHE_TTL: usize = 300; // 5 minutes

pub struct ConfidenceScorer {
    redis: Arc<RedisClient>,
    contradiction_decay: f64,
}

impl ConfidenceScorer {
    pub fn new(redis: Arc<RedisClient>, contradiction_decay: f64) -> Self {
        Self { redis, contradiction_decay }
    }

    /// Score an inbound CrisisEvent using Bayesian updating.
    /// Applies contradiction decay if contradictory cached signals are found.
    pub async fn score(&self, event: &CrisisEvent) -> Result<ConfidenceResult> {
        let severity_weight = SeverityWeights::weight(&event.severity);

        // Build signal reliability array from contributing event source tags
        // In production, contributing_events would carry source metadata;
        // here we use crisis_type-appropriate defaults.
        let signal_reliabilities = self.get_signal_reliabilities(event);

        // Bayesian update — prior is the CCA's raw confidence_score
        let updater = BayesianUpdater::new(event.confidence_score * severity_weight);
        let updated_confidence = updater.run(&signal_reliabilities);

        // Contradiction detection via Redis cache
        let (contradictions_detected, contradiction_sources, adjusted_confidence) =
            self.check_contradictions(event, updated_confidence).await;

        let is_actionable = adjusted_confidence >= ACTION_THRESHOLD;

        let classification = CrisisLabels::label(&event.crisis_type).to_string();

        if is_actionable {
            info!(
                crisis_id = %event.crisis_id,
                confidence = adjusted_confidence,
                classification = %classification,
                "crisis classified as actionable"
            );
        } else {
            warn!(
                crisis_id = %event.crisis_id,
                confidence = adjusted_confidence,
                "confidence below action threshold — not forwarding"
            );
        }

        Ok(ConfidenceResult {
            result_id: Uuid::new_v4().to_string(),
            crisis_id: event.crisis_id.clone(),
            hospital_id: event.hospital_id.clone(),
            final_confidence: updated_confidence,
            adjusted_confidence,
            classification,
            is_actionable,
            contradictions_detected,
            contradiction_sources,
            trace_id: event.trace_id.clone(),
            idempotency_key: Uuid::new_v4().to_string(),
            schema_version: event.schema_version.clone(),
            scored_at: Utc::now(),
        })
    }

    fn get_signal_reliabilities(&self, event: &CrisisEvent) -> Vec<f64> {
        use crate::models::crisis_event::CrisisType;
        match event.crisis_type {
            CrisisType::Ransomware =>
                vec![
                    SourceWeights::reliability("siem"),
                    SourceWeights::reliability("siem"),
                    SourceWeights::reliability("staff_report"),
                ],
            CrisisType::PowerFailure =>
                vec![
                    SourceWeights::reliability("grid_sensor"),
                    SourceWeights::reliability("grid_sensor"),
                ],
            CrisisType::IcuCapacityBreach =>
                vec![
                    SourceWeights::reliability("ehr"),
                    SourceWeights::reliability("ehr"),
                    SourceWeights::reliability("staff_report"),
                ],
            _ =>
                vec![SourceWeights::reliability("unknown")],
        }
    }

    async fn check_contradictions(
        &self,
        event: &CrisisEvent,
        base_confidence: f64,
    ) -> (bool, Vec<String>, f64) {
        let cache_key = format!(
            "{}{}:{}",
            CONTRADICTION_CACHE_PREFIX,
            event.hospital_id,
            event.crisis_id
        );

        // Cache the current crisis signal; look for opposing signals
        let _ = self
            .redis
            .set_ex(&cache_key, "active", CONTRADICTION_CACHE_TTL as u64)
            .await;

        // Check for a contradictory "all_clear" signal from the same hospital
        let clear_key = format!("all_clear:{}", event.hospital_id);
        let contradicted = self.redis.exists(&clear_key).await.unwrap_or(false);

        if contradicted {
            let decayed = base_confidence * self.contradiction_decay;
            warn!(
                crisis_id = %event.crisis_id,
                original = base_confidence,
                decayed,
                "contradiction detected — applying decay"
            );
            return (true, vec!["all_clear_signal".to_string()], decayed);
        }

        (false, vec![], base_confidence)
    }
}

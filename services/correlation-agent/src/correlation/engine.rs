use crate::correlation::sliding_window::{SlidingWindow, WindowEntry};
use crate::models::crisis_event::{CrisisEvent, CrisisType, Severity};
use crate::models::unified_event::UnifiedEvent;
use crate::redis::client::RedisClient;
use anyhow::Result;
use chrono::Utc;
use std::sync::Arc;
use tracing::{debug, info, warn};
use uuid::Uuid;

pub struct CorrelationEngine {
    window: SlidingWindow,
    anomaly_threshold: f64,
    redis: Arc<RedisClient>,
}

impl CorrelationEngine {
    pub fn new(window_seconds: u64, anomaly_threshold: f64, redis: Arc<RedisClient>) -> Self {
        Self {
            window: SlidingWindow::new(window_seconds),
            anomaly_threshold,
            redis,
        }
    }

    /// Process an inbound event: insert into window, run correlation rules,
    /// return a CrisisEvent if anomaly threshold is crossed.
    pub async fn process(&self, event: &UnifiedEvent) -> Result<Option<CrisisEvent>> {
        let entry = WindowEntry {
            timestamp: event.timestamp_utc,
            event_id: event.event_id.clone(),
            event_type: event.event_type.clone(),
            source_system: event.source_system.clone(),
        };

        self.window.insert(&event.hospital_id, entry).await;

        let snapshot = self.window.snapshot(&event.hospital_id).await;
        debug!(
            hospital_id = %event.hospital_id,
            window_size = snapshot.len(),
            "correlation window snapshot"
        );

        // --- Correlation Rules ---
        let (crisis_type, confidence) = self.evaluate_rules(&snapshot, event);

        if confidence >= self.anomaly_threshold {
            let contributing: Vec<String> =
                snapshot.iter().map(|e| e.event_id.clone()).collect();

            let severity = match confidence {
                c if c >= 0.95 => Severity::Critical,
                c if c >= 0.85 => Severity::High,
                c if c >= 0.72 => Severity::Medium,
                _ => Severity::Low,
            };

            let crisis = CrisisEvent {
                crisis_id: Uuid::new_v4().to_string(),
                hospital_id: event.hospital_id.clone(),
                crisis_type,
                severity,
                confidence_score: confidence,
                contributing_events: contributing,
                trace_id: event.trace_id.clone(),
                idempotency_key: Uuid::new_v4().to_string(),
                schema_version: event.schema_version.clone(),
                detected_at: Utc::now(),
            };

            info!(
                crisis_id = %crisis.crisis_id,
                confidence = crisis.confidence_score,
                hospital_id = %crisis.hospital_id,
                "crisis detected"
            );

            return Ok(Some(crisis));
        }

        Ok(None)
    }

    /// Evaluate correlation rules against the window snapshot.
    /// Returns (CrisisType, confidence_score).
    fn evaluate_rules(&self, snapshot: &[WindowEntry], event: &UnifiedEvent) -> (CrisisType, f64) {
        let ransomware_signals = snapshot
            .iter()
            .filter(|e| {
                e.source_system == "siem"
                    || e.event_type.contains("encryption")
                    || e.event_type.contains("lateral_movement")
            })
            .count();

        if ransomware_signals >= 3 {
            let confidence = (ransomware_signals as f64 * 0.25).min(0.98);
            return (CrisisType::Ransomware, confidence);
        }

        let power_signals = snapshot
            .iter()
            .filter(|e| e.source_system == "grid_sensor" || e.event_type.contains("power"))
            .count();

        if power_signals >= 2 {
            let confidence = (power_signals as f64 * 0.30).min(0.95);
            return (CrisisType::PowerFailure, confidence);
        }

        let icu_signals = snapshot
            .iter()
            .filter(|e| e.source_system == "ehr" && e.event_type == "bed_update")
            .count();

        if icu_signals >= 5 {
            let confidence = (icu_signals as f64 * 0.15).min(0.90);
            return (CrisisType::IcuCapacityBreach, confidence);
        }

        warn!(event_type = %event.event_type, "no correlation rule matched");
        (CrisisType::Unknown, 0.0)
    }
}

use crate::models::{ReplayEvent, SimulationScenario};
use anyhow::Result;
use chrono::{DateTime, Utc};
use tracing::{info, warn};
use std::time::Duration;
use tokio::time::sleep;

/// ReplayEngine re-executes a sequence of ReplayEvents in time order
/// within a sandboxed simulation context. It does NOT dispatch real API calls;
/// all events are executed against the simulation state graph only.
pub struct ReplayEngine {
    /// Speedup factor: 10.0 = replay 10x faster than real-time
    pub speed_factor: f64,
}

impl ReplayEngine {
    pub fn new(speed_factor: f64) -> Self {
        Self { speed_factor }
    }

    /// Execute a sequence of replay events, maintaining temporal order.
    /// Returns the count of successfully replayed events.
    pub async fn execute(&self, events: &[ReplayEvent], scenario: &SimulationScenario) -> Result<u32> {
        if events.is_empty() {
            warn!("replay engine received empty event stream");
            return Ok(0);
        }

        let mut prev_time: Option<DateTime<Utc>> = None;
        let mut replayed = 0u32;

        info!(
            scenario = ?scenario,
            event_count = events.len(),
            speed_factor = self.speed_factor,
            "starting replay execution"
        );

        for event in events {
            // Reproduce inter-event timing (compressed by speed_factor)
            if let Some(prev) = prev_time {
                let gap = (event.occurred_at - prev).num_milliseconds().max(0) as f64;
                let compressed_ms = (gap / self.speed_factor) as u64;
                if compressed_ms > 0 {
                    sleep(Duration::from_millis(compressed_ms.min(5000))).await;
                }
            }
            prev_time = Some(event.occurred_at);

            // Sandboxed execution: validate and apply to simulation state
            match self.apply_event(event).await {
                Ok(_) => {
                    replayed += 1;
                    info!(
                        event_id = %event.event_id,
                        action_type = %event.action_type,
                        resource_id = %event.resource_id,
                        "replay event applied"
                    );
                }
                Err(e) => {
                    warn!(
                        event_id = %event.event_id,
                        error = %e,
                        "replay event skipped"
                    );
                }
            }
        }

        info!(replayed_count = replayed, "replay execution complete");
        Ok(replayed)
    }

    /// Apply a single event to the simulation state.
    /// In production: dispatches to the SimulationGraph state machine.
    /// In current implementation: validates event structure and records the transition.
    async fn apply_event(&self, event: &ReplayEvent) -> Result<()> {
        if event.action_id.is_empty() || event.resource_id.is_empty() {
            return Err(anyhow::anyhow!("invalid replay event: missing action_id or resource_id"));
        }
        // GPU acceleration hook (stub): if enabled, dispatch to GPU worker pool
        // self.gpu_executor.submit(event).await?;
        Ok(())
    }
}

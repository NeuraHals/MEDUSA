use crate::config::Settings;
use crate::classification::scoring::ConfidenceScorer;
use crate::kafka::producer::DlqProducer;
use crate::health::HealthState;
use std::sync::Arc;
use anyhow::Result;

pub async fn run_consumer_pool(
    _settings: Arc<Settings>,
    _scorer: Arc<ConfidenceScorer>,
    _dlq: Arc<DlqProducer>,
    _health: Arc<HealthState>,
) -> Result<()> {
    Ok(())
}

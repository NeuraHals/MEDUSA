use anyhow::{Context, Result};
use config::{Config, Environment, File};
use serde::Deserialize;

#[derive(Debug, Deserialize, Clone)]
pub struct Settings {
    pub env: String,
    pub health_addr: String,
    pub kafka_brokers: Vec<String>,
    pub kafka_input_topic: String,
    pub kafka_output_topic: String,
    pub kafka_dlq_topic: String,
    pub kafka_consumer_group: String,
    pub kafka_worker_pool_size: usize,
    pub redis_url: String,
    pub otel_endpoint: String,
    /// Factor applied to confidence when contradictory signals are detected (0.0–1.0)
    pub contradiction_decay: f64,
    pub schema_version: String,
}

impl Settings {
    pub fn load() -> Result<Self> {
        let cfg = Config::builder()
            .add_source(File::with_name("config").required(false))
            .add_source(File::with_name("/config/config").required(false))
            .add_source(Environment::default().separator("__").try_parsing(true))
            .set_default("env", "production")?
            .set_default("health_addr", "0.0.0.0:8082")?
            .set_default("kafka_input_topic", "clinical.orchestration.anomaly.v1")?
            .set_default("kafka_output_topic", "clinical.orchestration.confidence.v1")?
            .set_default("kafka_dlq_topic", "system.dlq.v1")?
            .set_default("kafka_consumer_group", "confidence-group")?
            .set_default("kafka_worker_pool_size", 8)?
            .set_default("contradiction_decay", 0.60)?
            .set_default("schema_version", "1.0.0")?
            .build()
            .context("failed to build configuration")?;

        cfg.try_deserialize().context("failed to deserialise configuration")
    }
}

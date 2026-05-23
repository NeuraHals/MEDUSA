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
    pub schema_version: String,
    /// Redlock TTL in milliseconds for each resource lock
    pub lock_ttl_ms: u64,
    /// Maximum Redis graph traversal depth
    pub max_graph_depth: u32,
    /// Minimum confidence to generate an actionable blueprint
    pub action_confidence_threshold: f64,
    /// HOA gRPC endpoint for Tier 2 approval requests
    pub hoa_grpc_endpoint: String,
    /// Approval timeout in seconds
    pub approval_timeout_secs: u64,
}

impl Settings {
    pub fn load() -> Result<Self> {
        Config::builder()
            .add_source(File::with_name("config").required(false))
            .add_source(File::with_name("/config/config").required(false))
            .add_source(Environment::default().separator("__").try_parsing(true))
            .set_default("env", "production")?
            .set_default("health_addr", "0.0.0.0:8083")?
            .set_default("kafka_input_topic", "clinical.orchestration.confidence.v1")?
            .set_default("kafka_output_topic", "clinical.orchestration.blueprint.v1")?
            .set_default("kafka_dlq_topic", "system.dlq.v1")?
            .set_default("kafka_consumer_group", "allocation-group")?
            .set_default("kafka_worker_pool_size", 4)?
            .set_default("lock_ttl_ms", 5000_u64)?
            .set_default("max_graph_depth", 5)?
            .set_default("action_confidence_threshold", 0.72)?
            .set_default("approval_timeout_secs", 60_u64)?
            .set_default("schema_version", "1.0.0")?
            .build()
            .context("config build failed")?
            .try_deserialize()
            .context("config deserialise failed")
    }
}

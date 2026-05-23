use serde::Deserialize;
use config::{Config as ConfigLoader, ConfigError, Environment, File};

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    pub env: String,
    pub http_addr: String,
    pub kafka: KafkaConfig,
    pub redis: RedisConfig,
    pub otel: OtelConfig,
    pub schema_version: String,
    pub simulation: SimulationConfig,
}

#[derive(Debug, Deserialize, Clone)]
pub struct KafkaConfig {
    pub brokers: String,
    pub input_topic: String,
    pub output_topic: String,
    pub dlq_topic: String,
    pub consumer_group: String,
    pub worker_count: usize,
}

#[derive(Debug, Deserialize, Clone)]
pub struct RedisConfig {
    pub url: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct OtelConfig {
    pub endpoint: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct SimulationConfig {
    /// Default Monte Carlo iterations if not specified in request
    pub default_mc_runs: u32,
    /// Number of parallel simulation worker threads
    pub worker_threads: usize,
    /// Probability amplifier for chaos injection (0.0 - 1.0)
    pub chaos_factor: f64,
    /// Whether GPU acceleration hooks are enabled (stub)
    pub gpu_enabled: bool,
    /// Maximum simulation duration in seconds before timeout
    pub max_sim_secs: u64,
}

impl Config {
    pub fn load() -> Result<Self, ConfigError> {
        ConfigLoader::builder()
            .add_source(File::with_name("config").required(false))
            .add_source(File::with_name("/config/config").required(false))
            .add_source(Environment::with_prefix("SIPA").separator("__"))
            .build()?
            .try_deserialize()
    }
}

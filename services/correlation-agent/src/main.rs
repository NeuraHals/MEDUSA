use anyhow::Result;
use std::sync::Arc;
use tokio::signal;
use tracing::info;

mod config;
mod correlation;
mod health;
mod kafka;
mod logger;
mod middleware;
mod models;
mod otel;
mod redis;
mod state;

#[tokio::main]
async fn main() -> Result<()> {
    // Load config first — panics loudly if critical values are missing
    let cfg = Arc::new(config::Settings::load()?);

    // Init structured JSON logger
    logger::init(cfg.env.as_str());

    // Init OpenTelemetry tracer
    let _tracer = otel::init_tracer(&cfg.otel_endpoint, "correlation-agent")?;

    info!("correlation-agent starting");

    // Shared readiness flag
    let health = Arc::new(health::HealthState::new());

    // Redis connection manager
    let redis_client = Arc::new(
        redis::client::connect(&cfg.redis_url).await?,
    );

    // Sliding-window correlation engine
    let engine = Arc::new(
        correlation::engine::CorrelationEngine::new(
            cfg.window_seconds,
            cfg.anomaly_threshold,
            Arc::clone(&redis_client),
        ),
    );

    // Kafka DLQ producer (for failed events)
    let dlq_producer = Arc::new(
        kafka::producer::DlqProducer::new(&cfg.kafka_brokers, &cfg.kafka_dlq_topic),
    );

    // Health HTTP server (liveness + readiness)
    let health_handle = {
        let h = Arc::clone(&health);
        let addr = cfg.health_addr.clone();
        tokio::spawn(async move {
            health::server::run(addr, h).await;
        })
    };

    // Mark ready before spawning consumers
    health.set_ready(true);
    info!("correlation-agent ready");

    // Kafka consumer loop — bounded worker pool
    kafka::consumer::run_consumer_pool(
        Arc::clone(&cfg),
        Arc::clone(&engine),
        Arc::clone(&dlq_producer),
        Arc::clone(&health),
    )
    .await?;

    // Graceful shutdown on SIGINT/SIGTERM
    signal::ctrl_c().await?;
    info!("shutting down correlation-agent");
    health.set_ready(false);
    health_handle.abort();

    opentelemetry::global::shutdown_tracer_provider();
    Ok(())
}

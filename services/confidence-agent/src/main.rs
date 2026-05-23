use anyhow::Result;
use std::sync::Arc;
use tokio::signal;
use tracing::info;

mod classification;
mod config;
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
    let cfg = Arc::new(config::Settings::load()?);
    logger::init(cfg.env.as_str());

    let _tracer = otel::init_tracer(&cfg.otel_endpoint, "confidence-agent")?;
    info!("confidence-agent starting");

    let health = Arc::new(health::HealthState::new());

    let redis_client = Arc::new(
        redis::client::connect(&cfg.redis_url).await?,
    );

    let scorer = Arc::new(classification::scoring::ConfidenceScorer::new(
        Arc::clone(&redis_client),
        cfg.contradiction_decay,
    ));

    let dlq_producer = Arc::new(
        kafka::producer::DlqProducer::new(&cfg.kafka_brokers, &cfg.kafka_dlq_topic),
    );

    let health_handle = {
        let h = Arc::clone(&health);
        let addr = cfg.health_addr.clone();
        tokio::spawn(async move {
            health::server::run(addr, h).await;
        })
    };

    health.set_ready(true);
    info!("confidence-agent ready");

    kafka::consumer::run_consumer_pool(
        Arc::clone(&cfg),
        Arc::clone(&scorer),
        Arc::clone(&dlq_producer),
        Arc::clone(&health),
    )
    .await?;

    signal::ctrl_c().await?;
    info!("shutting down confidence-agent");
    health.set_ready(false);
    health_handle.abort();

    opentelemetry::global::shutdown_tracer_provider();
    Ok(())
}

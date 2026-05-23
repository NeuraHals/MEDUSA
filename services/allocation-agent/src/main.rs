use anyhow::Result;
use std::sync::Arc;
use tokio::signal;
use tracing::info;

mod allocation;
mod config;
mod grpc;
mod health;
mod kafka;
mod locking;
mod logger;
mod middleware;
mod models;
mod otel;
mod redis;
mod state;

#[tokio::main]
async fn main() -> Result<()> {
    let cfg = match config::Settings::load() {
        Ok(c) => Arc::new(c),
        Err(e) => {
            eprintln!("CRITICAL: Failed to load configuration: {}", e);
            std::process::exit(1);
        }
    };
    logger::init(cfg.env.as_str());
    info!("config loaded successfully");

    let _tracer = match otel::init_tracer(&cfg.otel_endpoint, "allocation-agent") {
        Ok(t) => t,
        Err(e) => {
            tracing::error!(error = %e, "Failed to initialize OpenTelemetry");
            std::process::exit(1);
        }
    };
    info!("allocation-agent starting");

    let health = Arc::new(health::HealthState::new());
    
    info!(redis_url = %cfg.redis_url, "Redis connecting");
    let redis_client = match redis::client::connect(&cfg.redis_url).await {
        Ok(c) => Arc::new(c),
        Err(e) => {
            tracing::error!(error = %e, "CRITICAL: Redis connection failed");
            std::process::exit(1);
        }
    };

    match redis_client.ping().await {
        Ok(_) => info!("Redis health check passed"),
        Err(e) => {
            tracing::error!(error = %e, "Redis health check failed");
            std::process::exit(1);
        }
    }

    let engine = Arc::new(allocation::engine::AllocationEngine::new(
        Arc::clone(&redis_client),
        cfg.lock_ttl_ms,
        cfg.max_graph_depth,
        cfg.action_confidence_threshold,
    ));

    info!(brokers = ?cfg.kafka_brokers, "Kafka OutputProducer connecting");
    let dlq_producer = match kafka::producer::OutputProducer::new(&cfg.kafka_brokers, &cfg.kafka_dlq_topic) {
        Ok(p) => {
            info!("Kafka producer ready");
            Arc::new(p)
        },
        Err(e) => {
            tracing::error!(error = %e, "CRITICAL: Kafka producer connection failed");
            std::process::exit(1);
        }
    };

    // Health server
    let health_handle = {
        let h = Arc::clone(&health);
        let addr = cfg.health_addr.clone();
        info!("HTTP health server listening on {}", addr);
        tokio::spawn(async move { health::server::run(addr, h).await })
    };

    health.set_ready(true);
    info!(
        kafka_brokers = ?cfg.kafka_brokers,
        redis_url = %cfg.redis_url,
        health_bind = %cfg.health_addr,
        "allocation-agent startup completed successfully"
    );

    info!("Kafka StreamConsumer connecting and subscribing");
    if let Err(e) = kafka::consumer::run_consumer_pool(
        Arc::clone(&cfg),
        Arc::clone(&engine),
        Arc::clone(&dlq_producer),
        Arc::clone(&health),
    )
    .await {
        tracing::error!(error = %e, "CRITICAL: Kafka consumer error");
        std::process::exit(1);
    }

    signal::ctrl_c().await?;
    info!("shutting down allocation-agent");
    health.set_ready(false);
    health_handle.abort();
    opentelemetry::global::shutdown_tracer_provider();
    Ok(())
}

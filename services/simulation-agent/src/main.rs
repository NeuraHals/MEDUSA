mod config;
mod health;
mod kafka;
mod middleware;
mod models;
mod montecarlo;
mod otel;
mod replay;
mod simulation;
mod state;
mod graph;

use std::sync::Arc;
use tokio::{signal, sync::watch};
use tracing::info;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Load configuration
    let cfg = config::Config::load().expect("Failed to load config");

    // Initialise OTel tracing
    otel::init_tracer(&cfg.otel.endpoint, "simulation-agent")?;

    // Redis state
    let redis = state::RedisState::new(&cfg.redis.url).await?;

    // Build Kafka producer
    let producer = Arc::new(kafka::build_producer(&cfg.kafka.brokers)?);

    // Build simulation orchestrator
    let orchestrator = Arc::new(simulation::SimulationOrchestrator::new(
        cfg.simulation.chaos_factor,
        redis.clone(),
        cfg.schema_version.clone(),
        cfg.simulation.max_sim_secs,
    ));

    let health = health::Health::new();

    // Graceful shutdown channel
    let (shutdown_tx, shutdown_rx) = watch::channel(false);

    // HTTP health server
    let health_clone = health.clone();
    let http_addr: std::net::SocketAddr = cfg.http_addr.parse()?;
    tokio::spawn(async move {
        let app = middleware::router(health_clone);
        info!("simulation-agent HTTP listening on {}", http_addr);
        let listener = tokio::net::TcpListener::bind(http_addr).await.unwrap();
        axum::serve(listener, app).await.unwrap();
    });

    // Kafka consumer loop
    let kafka_cfg = cfg.kafka.clone();
    let orch = orchestrator.clone();
    let redis_c = redis.clone();
    let prod = producer.clone();
    let schema_ver = cfg.schema_version.clone();
    let rx = shutdown_rx.clone();
    tokio::spawn(async move {
        kafka::start_consumer(kafka_cfg, orch, redis_c, prod, schema_ver, rx).await;
    });

    health.set_ready(true);
    info!(
        env = %cfg.env,
        http_addr = %cfg.http_addr,
        default_mc_runs = cfg.simulation.default_mc_runs,
        chaos_factor = cfg.simulation.chaos_factor,
        gpu_enabled = cfg.simulation.gpu_enabled,
        "simulation-agent ready"
    );

    // Await shutdown signal
    signal::ctrl_c().await?;
    info!("shutdown signal received");
    health.set_ready(false);
    let _ = shutdown_tx.send(true);

    otel::shutdown_tracer();
    info!("simulation-agent stopped");
    Ok(())
}

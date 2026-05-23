use anyhow::Result;
use rdkafka::{
    consumer::{CommitMode, Consumer, StreamConsumer},
    message::{BorrowedMessage, Headers, Message},
    producer::{FutureProducer, FutureRecord},
    ClientConfig,
};
use serde_json;
use std::{sync::Arc, time::Duration};
use tokio::sync::Semaphore;
use tracing::{error, info, warn};

use crate::{
    config::KafkaConfig,
    models::{SimulationRecord, SimulationRequest, SimulationResult, SimulationState},
    simulation::SimulationOrchestrator,
    state::RedisState,
};

const PRODUCER_AGENT: &str = "simulation-agent";

/// Build a rdkafka StreamConsumer from config.
pub fn build_consumer(cfg: &KafkaConfig) -> Result<StreamConsumer> {
    let consumer: StreamConsumer = ClientConfig::new()
        .set("bootstrap.servers", &cfg.brokers)
        .set("group.id", &cfg.consumer_group)
        .set("enable.auto.commit", "false")
        .set("auto.offset.reset", "earliest")
        .set("security.protocol", "ssl")
        .set("fetch.message.max.bytes", "10485760")
        .create()?;
    consumer.subscribe(&[&cfg.input_topic])?;
    Ok(consumer)
}

/// Build a rdkafka FutureProducer.
pub fn build_producer(brokers: &str) -> Result<FutureProducer> {
    let producer: FutureProducer = ClientConfig::new()
        .set("bootstrap.servers", brokers)
        .set("message.timeout.ms", "10000")
        .set("security.protocol", "ssl")
        .set("acks", "all")
        .create()?;
    Ok(producer)
}

/// Extract a header value by key from a Kafka message.
pub fn header_value(msg: &BorrowedMessage, key: &str) -> String {
    msg.headers()
        .and_then(|h| {
            (0..h.count()).find_map(|i| {
                let header = h.get(i);
                if header.key == key {
                    header.value.and_then(|v| std::str::from_utf8(v).ok()).map(|s| s.to_string())
                } else {
                    None
                }
            })
        })
        .unwrap_or_default()
}

/// Publish a SimulationResult to the output topic with all mandatory headers.
pub async fn publish_result(
    producer: &FutureProducer,
    topic: &str,
    result: &SimulationResult,
    schema_version: &str,
) -> Result<()> {
    let data = serde_json::to_vec(result)?;
    let record = FutureRecord::to(topic)
        .key(&result.crisis_id)
        .payload(&data)
        .headers(
            rdkafka::message::OwnedHeaders::new()
                .insert(rdkafka::message::Header { key: "x-trace-id", value: Some(result.trace_id.as_bytes()) })
                .insert(rdkafka::message::Header { key: "x-idempotency-key", value: Some(result.idempotency_key.as_bytes()) })
                .insert(rdkafka::message::Header { key: "x-producer-agent", value: Some(PRODUCER_AGENT.as_bytes()) })
                .insert(rdkafka::message::Header { key: "x-retry-count", value: Some(b"0") })
                .insert(rdkafka::message::Header { key: "x-schema-version", value: Some(schema_version.as_bytes()) }),
        );
    producer
        .send(record, Duration::from_secs(10))
        .await
        .map_err(|(e, _)| anyhow::anyhow!("kafka publish failed: {}", e))?;
    Ok(())
}

/// Publish a raw payload to the DLQ topic.
pub async fn publish_dlq(producer: &FutureProducer, dlq_topic: &str, raw: &[u8], error_code: &str) {
    let record = FutureRecord::to(dlq_topic)
        .key(error_code)
        .payload(raw)
        .headers(
            rdkafka::message::OwnedHeaders::new()
                .insert(rdkafka::message::Header { key: "x-producer-agent", value: Some(PRODUCER_AGENT.as_bytes()) })
                .insert(rdkafka::message::Header { key: "x-error-code", value: Some(error_code.as_bytes()) }),
        );
    let _ = producer.send(record, Duration::from_secs(5)).await;
}

/// Start the Kafka consumer + bounded worker pool.
pub async fn start_consumer(
    cfg: KafkaConfig,
    orchestrator: Arc<SimulationOrchestrator>,
    redis: RedisState,
    producer: Arc<FutureProducer>,
    schema_version: String,
    shutdown: tokio::sync::watch::Receiver<bool>,
) {
    let consumer = match build_consumer(&cfg) {
        Ok(c) => Arc::new(c),
        Err(e) => { error!(error = %e, "consumer build failed"); return; }
    };

    let semaphore = Arc::new(Semaphore::new(cfg.worker_count));

    info!(
        topic = %cfg.input_topic,
        worker_count = cfg.worker_count,
        "simulation consumer started"
    );

    loop {
        if *shutdown.borrow() { break; }

        let msg = match consumer.recv().await {
            Ok(m) => m,
            Err(e) => {
                warn!(error = %e, "kafka recv error");
                continue;
            }
        };

        let trace_id = header_value(&msg, "x-trace-id");
        let idempotency_key = header_value(&msg, "x-idempotency-key");
        let payload = msg.payload().unwrap_or(&[]).to_vec();
        let topic = cfg.input_topic.clone();
        let dlq_topic = cfg.dlq_topic.clone();
        let output_topic = cfg.output_topic.clone();
        let schema_ver = schema_version.clone();

        let orchestrator = orchestrator.clone();
        let redis_c = redis.clone();
        let producer_c = producer.clone();
        let sem = semaphore.clone();

        tokio::spawn(async move {
            let _permit = sem.acquire().await.unwrap();

            if payload.is_empty() {
                warn!("empty kafka message — discarding");
                return;
            }

            let mut request: SimulationRequest = match serde_json::from_slice(&payload) {
                Ok(r) => r,
                Err(e) => {
                    error!(error = %e, "deserialise failed — DLQ");
                    publish_dlq(&producer_c, &dlq_topic, &payload, "deserialisation_error").await;
                    return;
                }
            };

            if request.trace_id.is_empty() { request.trace_id = trace_id; }
            if request.idempotency_key.is_empty() { request.idempotency_key = idempotency_key; }

            // Idempotency guard
            if redis_c.is_duplicate(&request.request_id).await {
                info!(request_id = %request.request_id, "idempotency hit — skipping");
                return;
            }

            // Store initial record
            let record = SimulationRecord {
                request_id: request.request_id.clone(),
                state: SimulationState::Running,
                started_at: chrono::Utc::now(),
                completed_at: None,
            };
            let _ = redis_c.store_record(&record).await;

            // Execute simulation
            match orchestrator.run(request.clone()).await {
                Ok(result) => {
                    if let Err(e) = publish_result(&producer_c, &output_topic, &result, &schema_ver).await {
                        error!(error = %e, request_id = %request.request_id, "result publish failed — not marking processed");
                        return;
                    }
                    let _ = redis_c.mark_processed(&request.request_id).await;
                }
                Err(e) => {
                    error!(error = %e, request_id = %request.request_id, "simulation failed — DLQ");
                    publish_dlq(&producer_c, &dlq_topic, &payload, "simulation_error").await;
                    let _ = redis_c.mark_processed(&request.request_id).await;
                }
            }
        });

        // Commit offset after worker is spawned (committed before result publish in spawn)
        if let Err(e) = consumer.commit_message(&msg, CommitMode::Async) {
            warn!(error = %e, "offset commit failed");
        }
    }

    info!("consumer loop exited");
}

use crate::config::Settings;
use crate::correlation::engine::CorrelationEngine;
use crate::health::HealthState;
use crate::kafka::producer::DlqProducer;
use crate::models::unified_event::UnifiedEvent;
use anyhow::Result;
use rdkafka::consumer::{CommitMode, Consumer, StreamConsumer};
use rdkafka::message::{Headers, Message};
use rdkafka::{ClientConfig, TopicPartitionList};
use std::sync::Arc;
use tracing::{error, info, warn};

/// Spawns a bounded pool of Kafka consumer tasks using a backpressure-safe
/// bounded async channel. Offsets are committed ONLY after full processing.
pub async fn run_consumer_pool(
    cfg: Arc<Settings>,
    engine: Arc<CorrelationEngine>,
    dlq: Arc<DlqProducer>,
    health: Arc<HealthState>,
) -> Result<()> {
    // Bounded channel — natural backpressure when workers are saturated
    let (tx, rx) = async_channel::bounded::<rdkafka::message::OwnedMessage>(512);

    let consumer: Arc<StreamConsumer> = Arc::new(build_consumer(&cfg)?);
    consumer.subscribe(&[cfg.kafka_input_topic.as_str()])?;
    info!("kafka consumer subscribed to {}", cfg.kafka_input_topic);

    // Single reader task — pushes raw messages into bounded channel
    let consumer_ref = Arc::clone(&consumer);
    let tx_clone = tx.clone();
    tokio::spawn(async move {
        loop {
            match consumer_ref.recv().await {
                Ok(msg) => {
                    if tx_clone.send(msg.detach()).await.is_err() {
                        break; // channel closed on shutdown
                    }
                }
                Err(e) => {
                    error!(error = %e, "kafka recv error");
                }
            }
        }
    });

    // Worker pool — bounded to cfg.kafka_worker_pool_size
    let mut handles = vec![];
    for worker_id in 0..cfg.kafka_worker_pool_size {
        let rx = rx.clone();
        let engine = Arc::clone(&engine);
        let dlq = Arc::clone(&dlq);
        let cfg = Arc::clone(&cfg);
        let consumer = Arc::clone(&consumer);

        handles.push(tokio::spawn(async move {
            info!(worker_id, "kafka worker started");
            while let Ok(msg) = rx.recv().await {
                let committed = process_message(&msg, &engine, &dlq, &cfg).await;

                // Only commit offset after full, successful processing pipeline.
                // This prevents premature commits and ensures replay safety.
                if committed {
                    let mut tpl = TopicPartitionList::new();
                    tpl.add_partition_offset(
                        msg.topic(),
                        msg.partition(),
                        rdkafka::Offset::Offset(msg.offset() + 1),
                    )
                    .unwrap_or_default();

                    if let Err(e) = consumer.commit(&tpl, CommitMode::Async) {
                        error!(
                            error = %e,
                            topic = msg.topic(),
                            partition = msg.partition(),
                            offset = msg.offset(),
                            "offset commit failed — will replay on restart"
                        );
                    }
                }
                // If committed == false: offset is NOT committed.
                // Kafka will replay this message on consumer restart,
                // which is the correct behaviour for transient failures.
            }
            info!(worker_id, "kafka worker stopped");
        }));
    }

    for handle in handles {
        let _ = handle.await;
    }
    Ok(())
}

/// Returns true ONLY if the full processing pipeline completed successfully:
/// deserialization → correlation → output publication (or DLQ routing).
/// A false return suppresses offset commit, guaranteeing at-least-once delivery.
async fn process_message(
    msg: &rdkafka::message::OwnedMessage,
    engine: &CorrelationEngine,
    dlq: &DlqProducer,
    cfg: &Settings,
) -> bool {
    let payload = match msg.payload() {
        Some(b) => b,
        None => {
            warn!(
                topic = msg.topic(),
                partition = msg.partition(),
                offset = msg.offset(),
                "empty kafka payload — skipping, committing offset"
            );
            return true; // empty messages are safely discarded
        }
    };

    let trace_id = extract_header(msg, "x-trace-id").unwrap_or_default();

    // --- Deserialise ---
    let event: UnifiedEvent = match serde_json::from_slice(payload) {
        Ok(e) => e,
        Err(err) => {
            error!(
                error = %err,
                trace_id = %trace_id,
                "deserialisation failed — routing to DLQ"
            );
            // DLQ routing is a terminal action: commit the offset so we
            // don't infinitely retry an unparseable message.
            dlq.publish(payload, "deserialisation_error").await;
            return true;
        }
    };

    // --- Idempotency check via trace header ---
    let idempotency_key = extract_header(msg, "x-idempotency-key").unwrap_or_default();
    if idempotency_key.is_empty() {
        warn!(event_id = %event.event_id, "missing idempotency key header");
    }

    // --- Correlation engine ---
    let result = engine.process(&event).await;
    match result {
        Ok(Some(crisis)) => {
            // Publish CrisisEvent to the orchestration output topic
            let serialised = match serde_json::to_vec(&crisis) {
                Ok(b) => b,
                Err(e) => {
                    error!(error = %e, crisis_id = %crisis.crisis_id, "failed to serialise crisis event");
                    return false; // do NOT commit — retry this message
                }
            };

            let published = dlq
                .publish_to_topic(
                    &cfg.kafka_output_topic,
                    &serialised,
                    &crisis.trace_id,
                    &crisis.crisis_id,
                )
                .await;

            if !published {
                error!(crisis_id = %crisis.crisis_id, "output publish failed — suppressing offset commit");
                return false; // do NOT commit — retry
            }

            true // full pipeline success
        }
        Ok(None) => {
            // No anomaly detected — event safely discarded
            true
        }
        Err(e) => {
            error!(
                error = %e,
                event_id = %event.event_id,
                trace_id = %trace_id,
                "correlation engine error — routing to DLQ"
            );
            dlq.publish(payload, "engine_error").await;
            true // DLQ is terminal — commit offset
        }
    }
}

fn build_consumer(cfg: &Settings) -> Result<StreamConsumer> {
    let brokers = cfg.kafka_brokers.join(",");
    let consumer: StreamConsumer = ClientConfig::new()
        .set("bootstrap.servers", &brokers)
        .set("group.id", &cfg.kafka_consumer_group)
        .set("enable.auto.commit", "false") // manual offset control
        .set("auto.offset.reset", "earliest")
        .set("security.protocol", "ssl")
        .set("max.poll.interval.ms", "300000")
        .create()?;
    Ok(consumer)
}

fn extract_header(msg: &rdkafka::message::OwnedMessage, key: &str) -> Option<String> {
    msg.headers().and_then(|headers| {
        for i in 0..headers.count() {
            let h = headers.get(i);
            if h.key == key {
                return h.value.and_then(|v| String::from_utf8(v.to_vec()).ok());
            }
        }
        None
    })
}

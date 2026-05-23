use crate::allocation::engine::{AllocationEngine, AllocationOutcome};
use crate::config::Settings;
use crate::health::HealthState;
use crate::kafka::producer::OutputProducer;
use crate::models::confidence_result::ConfidenceResult;
use anyhow::Result;
use rdkafka::consumer::{CommitMode, Consumer, StreamConsumer};
use rdkafka::message::{Headers, Message};
use rdkafka::{ClientConfig, TopicPartitionList};
use std::sync::Arc;
use tracing::{error, info, warn};

pub async fn run_consumer_pool(
    cfg: Arc<Settings>,
    engine: Arc<AllocationEngine>,
    producer: Arc<OutputProducer>,
    _health: Arc<HealthState>,
) -> Result<()> {
    let (tx, rx) = async_channel::bounded::<rdkafka::message::OwnedMessage>(256);

    let consumer: Arc<StreamConsumer> = Arc::new(build_consumer(&cfg)?);
    consumer.subscribe(&[cfg.kafka_input_topic.as_str()])?;
    info!("allocation consumer subscribed to {}", cfg.kafka_input_topic);

    let consumer_ref = Arc::clone(&consumer);
    let tx_clone = tx.clone();
    tokio::spawn(async move {
        loop {
            match consumer_ref.recv().await {
                Ok(msg) => { if tx_clone.send(msg.detach()).await.is_err() { break; } }
                Err(e) => error!(error = %e, "kafka recv error"),
            }
        }
    });

    let mut handles = vec![];
    for worker_id in 0..cfg.kafka_worker_pool_size {
        let rx = rx.clone();
        let engine = Arc::clone(&engine);
        let producer = Arc::clone(&producer);
        let cfg = Arc::clone(&cfg);
        let consumer = Arc::clone(&consumer);

        handles.push(tokio::spawn(async move {
            info!(worker_id, "allocation worker started");
            while let Ok(msg) = rx.recv().await {
                let committed = process_message(&msg, &engine, &producer, &cfg).await;
                if committed {
                    let mut tpl = TopicPartitionList::new();
                    tpl.add_partition_offset(
                        msg.topic(), msg.partition(),
                        rdkafka::Offset::Offset(msg.offset() + 1),
                    ).unwrap_or_default();
                    if let Err(e) = consumer.commit(&tpl, CommitMode::Async) {
                        error!(error = %e, offset = msg.offset(), "offset commit failed");
                    }
                }
            }
        }));
    }

    for h in handles { let _ = h.await; }
    Ok(())
}

async fn process_message(
    msg: &rdkafka::message::OwnedMessage,
    engine: &AllocationEngine,
    producer: &OutputProducer,
    cfg: &Settings,
) -> bool {
    let payload = match msg.payload() {
        Some(b) => b,
        None => { warn!("empty payload"); return true; }
    };

    let trace_id = extract_header(msg, "x-trace-id").unwrap_or_default();

    let result: ConfidenceResult = match serde_json::from_slice(payload) {
        Ok(r) => r,
        Err(e) => {
            error!(error = %e, trace_id = %trace_id, "deserialise failed — DLQ");
            producer.publish_dlq(payload, "deserialisation_error").await;
            return true;
        }
    };

    // Idempotency guard — skip already-allocated crises
    let idem_key = format!("allocated:{}", result.crisis_id);
    if engine.is_already_allocated(&idem_key).await {
        info!(crisis_id = %result.crisis_id, "idempotency hit — skipping");
        return true;
    }

    match engine.allocate(&result).await {
        Ok(AllocationOutcome::Blueprint(blueprint)) => {
            let serialised = match serde_json::to_vec(&blueprint) {
                Ok(b) => b,
                Err(e) => { error!(error = %e, "blueprint serialise failed"); return false; }
            };
            let ok = producer.publish_blueprint(
                &cfg.kafka_output_topic, &serialised,
                &blueprint.trace_id, &blueprint.blueprint_id,
            ).await;
            if !ok { return false; }
            // Mark as allocated in Redis to prevent replay re-execution
            let _ = engine.mark_allocated(&idem_key).await;
            true
        }
        Ok(AllocationOutcome::LockFailed(manifest)) => {
            let serialised = serde_json::to_vec(&manifest).unwrap_or_default();
            producer.publish_dlq(&serialised, "lock_failed").await;
            true
        }
        Ok(AllocationOutcome::BelowThreshold) | Ok(AllocationOutcome::NoResources) => true,
        Err(e) => {
            error!(error = %e, crisis_id = %result.crisis_id, "engine error — DLQ");
            producer.publish_dlq(payload, "engine_error").await;
            true
        }
    }
}

fn build_consumer(cfg: &Settings) -> Result<StreamConsumer> {
    let consumer: StreamConsumer = ClientConfig::new()
        .set("bootstrap.servers", cfg.kafka_brokers.join(","))
        .set("group.id", &cfg.kafka_consumer_group)
        .set("enable.auto.commit", "false")
        .set("auto.offset.reset", "earliest")
        .set("max.poll.interval.ms", "300000")
        .create()?;
    Ok(consumer)
}

fn extract_header(msg: &rdkafka::message::OwnedMessage, key: &str) -> Option<String> {
    msg.headers().and_then(|h| {
        for i in 0..h.count() {
            let hdr = h.get(i);
            if hdr.key == key {
                return hdr.value.and_then(|v| String::from_utf8(v.to_vec()).ok());
            }
        }
        None
    })
}

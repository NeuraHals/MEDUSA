use anyhow::Result;
use rdkafka::producer::{FutureProducer, FutureRecord};
use rdkafka::ClientConfig;
use std::time::Duration;
use tracing::error;

pub struct OutputProducer {
    producer: FutureProducer,
    dlq_topic: String,
}

impl OutputProducer {
    pub fn new(brokers: &[String], dlq_topic: &str) -> Result<Self> {
        let producer: FutureProducer = ClientConfig::new()
            .set("bootstrap.servers", brokers.join(","))
            .set("message.timeout.ms", "5000")
            .create()?;
        Ok(Self { producer, dlq_topic: dlq_topic.to_string() })
    }

    /// Publish a blueprint to the orchestration bus. Returns true on success.
    pub async fn publish_blueprint(
        &self, topic: &str, payload: &[u8], trace_id: &str, key: &str,
    ) -> bool {
        let record = FutureRecord::to(topic).payload(payload).key(key);
        match self.producer.send(record, Duration::from_secs(2)).await {
            Ok(_) => true,
            Err((e, _)) => {
                error!(error = %e, trace_id = %trace_id, topic, "blueprint publish failed");
                false
            }
        }
    }

    /// Route a failed or rolled-back payload to the DLQ.
    pub async fn publish_dlq(&self, payload: &[u8], error_code: &str) {
        let record = FutureRecord::to(&self.dlq_topic).payload(payload).key(error_code);
        if let Err((e, _)) = self.producer.send(record, Duration::from_secs(2)).await {
            error!(error = %e, "DLQ publish failed");
        }
    }
}

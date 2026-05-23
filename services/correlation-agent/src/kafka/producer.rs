use anyhow::Result;
use rdkafka::producer::{FutureProducer, FutureRecord};
use rdkafka::ClientConfig;
use std::time::Duration;
use tracing::error;

pub struct DlqProducer {
    producer: FutureProducer,
    dlq_topic: String,
}

impl DlqProducer {
    pub fn new(brokers: &[String], dlq_topic: &str) -> Self {
        let producer: FutureProducer = ClientConfig::new()
            .set("bootstrap.servers", brokers.join(","))
            .set("message.timeout.ms", "5000")
            .set("security.protocol", "ssl")
            .create()
            .expect("DLQ producer creation failed");
        Self { producer, dlq_topic: dlq_topic.to_string() }
    }

    /// Route a failed payload to the DLQ topic. Always returns — never panics.
    pub async fn publish(&self, payload: &[u8], error_code: &str) {
        let record = FutureRecord::to(&self.dlq_topic)
            .payload(payload)
            .key(error_code);
        if let Err((e, _)) = self.producer.send(record, Duration::from_secs(2)).await {
            error!(error = %e, "DLQ publish failed — message may be lost");
        }
    }

    /// Publish to an arbitrary output topic. Returns true on success.
    pub async fn publish_to_topic(
        &self,
        topic: &str,
        payload: &[u8],
        trace_id: &str,
        key: &str,
    ) -> bool {
        let record = FutureRecord::to(topic).payload(payload).key(key);
        match self.producer.send(record, Duration::from_secs(2)).await {
            Ok(_) => true,
            Err((e, _)) => {
                error!(error = %e, trace_id = %trace_id, topic = topic, "output publish failed");
                false
            }
        }
    }
}

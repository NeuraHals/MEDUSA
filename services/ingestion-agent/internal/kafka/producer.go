package kafka

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
)

const producerAgent = "ingestion-agent"

// Producer wraps kafka-go with mandatory header injection.
type Producer struct {
	writer        *kafkago.Writer
	schemaVersion string
}

// NewProducer creates a resilient, retry-safe Kafka writer.
func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{}, // partition by key (hospital_id)
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafkago.RequireAll,
			MaxAttempts:  3,
			Async:        false,
		},
	}
}

// Publish serialises the payload and attaches all mandatory Kafka headers.
// It derives the IdempotencyKey as SHA-256(eventID) to ensure replay safety.
func (p *Producer) Publish(ctx context.Context, payload []byte, eventID string) error {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	idempotencyKey := fmt.Sprintf("%x", sha256.Sum256([]byte(eventID)))

	msg := kafkago.Message{
		Key:   []byte(eventID),
		Value: payload,
		Headers: []kafkago.Header{
			{Key: "x-trace-id", Value: []byte(traceID)},
			{Key: "x-idempotency-key", Value: []byte(idempotencyKey)},
			{Key: "x-producer-agent", Value: []byte(producerAgent)},
			{Key: "x-schema-version", Value: []byte(p.schemaVersion)},
		},
	}

	return p.writer.WriteMessages(ctx, msg)
}

// Close flushes and closes the underlying writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}

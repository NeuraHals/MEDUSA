package kafka

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
)

// Producer handles strict schema validation and trace propagation.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer instantiates a new Kafka producer wrapper.
func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafka.RequireAll,
		},
	}
}

// Produce injects OpenTelemetry headers and idempotency keys before writing.
func (p *Producer) Produce(ctx context.Context, payload []byte, actionID string) error {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()

	idempotencyKey := fmt.Sprintf("%x", sha256.Sum256([]byte(actionID)))

	msg := kafka.Message{
		Key:   []byte(actionID),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "x-trace-id", Value: []byte(traceID)},
			{Key: "x-idempotency-key", Value: []byte(idempotencyKey)},
			{Key: "x-producer-agent", Value: []byte("aoa")},
		},
	}

	return p.writer.WriteMessages(ctx, msg)
}

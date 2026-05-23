package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
)

const producerAgent = "orchestrator-agent"

// Producer publishes execution events and DLQ messages.
type Producer struct {
	writer    *kafkago.Writer
	dlqTopic  string
	schemaVer string
}

func NewProducer(brokers []string, outputTopic string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        outputTopic,
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireAll,
			MaxAttempts:  3,
		},
	}
}

// Publish sends a typed payload to the output topic with mandatory headers.
func (p *Producer) Publish(ctx context.Context, payload interface{}, blueprintID, traceID, idempotencyKey string) error {
	span := trace.SpanFromContext(ctx)
	if traceID == "" {
		traceID = span.SpanContext().TraceID().String()
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	return p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(blueprintID),
		Value: data,
		Headers: []kafkago.Header{
			{Key: "x-trace-id", Value: []byte(traceID)},
			{Key: "x-idempotency-key", Value: []byte(idempotencyKey)},
			{Key: "x-producer-agent", Value: []byte(producerAgent)},
			{Key: "x-schema-version", Value: []byte(p.schemaVer)},
			{Key: "x-retry-count", Value: []byte("0")},
		},
	})
}

// PublishDLQ routes a failed raw payload to the DLQ topic.
func (p *Producer) PublishDLQ(ctx context.Context, raw []byte, errorCode string) {
	msg := kafkago.Message{
		Topic: p.dlqTopic,
		Key:   []byte(errorCode),
		Value: raw,
		Headers: []kafkago.Header{
			{Key: "x-producer-agent", Value: []byte(producerAgent)},
			{Key: "x-error-code", Value: []byte(errorCode)},
		},
	}
	writer := &kafkago.Writer{
		Addr:  p.writer.Addr,
		Topic: p.dlqTopic,
	}
	defer writer.Close()
	_ = writer.WriteMessages(ctx, msg)
}

func (p *Producer) Close() { _ = p.writer.Close() }

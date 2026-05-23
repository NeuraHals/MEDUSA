package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
)

const producerAgent = "oversight-agent"

// Producer publishes approval decision events to the orchestration bus.
type Producer struct {
	writer      *kafkago.Writer
	dlqTopic    string
	schemaVer   string
}

func NewProducer(brokers []string, outputTopic, dlqTopic, schemaVer string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        outputTopic,
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireAll,
			MaxAttempts:  3,
		},
		dlqTopic:  dlqTopic,
		schemaVer: schemaVer,
	}
}

// PublishDecision sends an approval decision event with all mandatory headers.
func (p *Producer) PublishDecision(
	ctx context.Context,
	payload interface{},
	blueprintID, traceID, idempotencyKey string,
) error {
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
			{Key: "x-retry-count", Value: []byte("0")},
			{Key: "x-schema-version", Value: []byte(p.schemaVer)},
		},
	})
}

// PublishDLQ routes failed payloads to the DLQ.
func (p *Producer) PublishDLQ(ctx context.Context, raw []byte, errorCode string) {
	w := &kafkago.Writer{Addr: p.writer.Addr, Topic: p.dlqTopic}
	defer w.Close()
	_ = w.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(errorCode),
		Value: raw,
		Headers: []kafkago.Header{
			{Key: "x-producer-agent", Value: []byte(producerAgent)},
			{Key: "x-error-code", Value: []byte(errorCode)},
		},
	})
}

func (p *Producer) Close() { _ = p.writer.Close() }

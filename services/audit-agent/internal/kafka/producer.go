package kafka

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
)

const producerAgent = "audit-agent"

// Producer routes failed audit events to the DLQ.
// The ACA does not publish forward events — it is a terminal sink.
type Producer struct {
	dlqTopic string
	brokers  []string
}

func NewProducer(brokers []string, dlqTopic string) *Producer {
	return &Producer{dlqTopic: dlqTopic, brokers: brokers}
}

func (p *Producer) PublishDLQ(ctx context.Context, raw []byte, errorCode string) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()

	w := &kafkago.Writer{
		Addr:  kafkago.TCP(p.brokers...),
		Topic: p.dlqTopic,
	}
	defer w.Close()
	_ = w.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(errorCode),
		Value: raw,
		Headers: []kafkago.Header{
			{Key: "x-producer-agent", Value: []byte(producerAgent)},
			{Key: "x-error-code", Value: []byte(errorCode)},
			{Key: "x-trace-id", Value: []byte(traceID)},
		},
	})
}

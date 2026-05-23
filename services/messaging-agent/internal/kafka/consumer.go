package kafka

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/antigravity/mono/services/messaging-agent/internal/config"
	"github.com/antigravity/mono/services/messaging-agent/internal/health"
	"github.com/antigravity/mono/services/messaging-agent/internal/messaging"
	"github.com/antigravity/mono/services/messaging-agent/internal/models"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var consumerTracer = otel.Tracer("messaging-agent/consumer")

// Consumer drives the notification worker pool.
type Consumer struct {
	cfg        *config.Config
	log        *zap.Logger
	dispatcher *messaging.Dispatcher
	producer   *Producer
	health     *health.Health
	reader     *kafkago.Reader
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func NewConsumer(
	cfg *config.Config,
	log *zap.Logger,
	dispatcher *messaging.Dispatcher,
	producer *Producer,
	hc *health.Health,
) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		Topic:          cfg.KafkaInputTopic,
		GroupID:        cfg.KafkaConsumerGroup,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // manual commit
	})
	return &Consumer{
		cfg:        cfg,
		log:        log,
		dispatcher: dispatcher,
		producer:   producer,
		health:     hc,
		reader:     reader,
		stopCh:     make(chan struct{}),
	}
}

func (c *Consumer) Start(ctx context.Context) {
	msgCh := make(chan kafkago.Message, 256)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				select {
				case <-c.stopCh:
					return
				default:
					c.log.Error("kafka read error", zap.Error(err))
					time.Sleep(500 * time.Millisecond)
					continue
				}
			}
			msgCh <- msg
		}
	}()

	for i := 0; i < c.cfg.KafkaWorkerCount; i++ {
		workerID := i
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.log.Info("messaging worker started", zap.Int("worker_id", workerID))
			for {
				select {
				case <-c.stopCh:
					return
				case msg := <-msgCh:
					committed := c.processMessage(ctx, msg)
					if committed {
						if err := c.reader.CommitMessages(ctx, msg); err != nil {
							c.log.Error("offset commit failed",
								zap.Error(err), zap.Int64("offset", msg.Offset),
							)
						}
					}
				}
			}
		}()
	}
}

func (c *Consumer) Stop() {
	close(c.stopCh)
	_ = c.reader.Close()
	c.wg.Wait()
}

// processMessage deserialises, dispatches, publishes results, and returns whether to commit.
func (c *Consumer) processMessage(ctx context.Context, msg kafkago.Message) bool {
	traceID := headerValue(msg.Headers, "x-trace-id")
	idempotencyKey := headerValue(msg.Headers, "x-idempotency-key")

	ctx, span := consumerTracer.Start(ctx, "consumer.processMessage")
	defer span.End()
	span.SetAttributes(attribute.String("trace.id", traceID))

	if len(msg.Value) == 0 {
		c.log.Warn("empty kafka message — discarding")
		return true
	}

	var req models.NotificationRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		c.log.Error("deserialisation failed — DLQ",
			zap.Error(err), zap.String("trace_id", traceID),
		)
		c.producer.PublishDLQ(ctx, msg.Value, "deserialisation_error")
		return true // terminal — commit
	}

	// Propagate headers into model if not set by upstream
	if req.TraceID == "" {
		req.TraceID = traceID
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = idempotencyKey
	}

	results := c.dispatcher.Dispatch(ctx, &req)

	// Publish delivery result events
	for _, result := range results {
		if err := c.producer.PublishResult(ctx, &result, req.CrisisID, req.TraceID, req.IdempotencyKey); err != nil {
			c.log.Error("result publish failed — suppressing commit",
				zap.Error(err), zap.String("request_id", req.RequestID),
			)
			return false // do not commit — retry the batch
		}
	}

	return true
}

func headerValue(headers []kafkago.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

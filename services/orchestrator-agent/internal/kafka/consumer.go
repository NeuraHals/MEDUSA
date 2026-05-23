package kafka

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/antigravity/mono/services/orchestrator-agent/internal/config"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/execution"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/health"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/models"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/state"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var consumerTracer = otel.Tracer("orchestrator-agent/consumer")

// Consumer manages the Kafka consumer pool for blueprint events.
type Consumer struct {
	cfg      *config.Config
	log      *zap.Logger
	producer *Producer
	redis    *state.RedisClient
	health   *health.Health
	engine   *execution.Engine
	reader   *kafkago.Reader
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewConsumer(
	cfg *config.Config,
	log *zap.Logger,
	producer *Producer,
	redis *state.RedisClient,
	hc *health.Health,
) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        cfg.KafkaBrokers,
		Topic:          cfg.KafkaInputTopic,
		GroupID:        cfg.KafkaConsumerGroup,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // manual commit only
	})

	eng := execution.NewEngine(log, redis, cfg.SchemaVersion)

	return &Consumer{
		cfg:      cfg,
		log:      log,
		producer: producer,
		redis:    redis,
		health:   hc,
		engine:   eng,
		reader:   reader,
		stopCh:   make(chan struct{}),
	}
}

// Start launches a bounded worker pool and a single reader goroutine.
func (c *Consumer) Start(ctx context.Context) {
	msgCh := make(chan kafkago.Message, 256)

	// Reader goroutine
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

	// Worker pool
	for i := 0; i < c.cfg.KafkaWorkerCount; i++ {
		workerID := i
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.log.Info("orchestrator worker started", zap.Int("worker_id", workerID))
			for {
				select {
				case <-c.stopCh:
					return
				case msg := <-msgCh:
					c.processMessage(ctx, msg)
					// Commit offset after full processing
					if err := c.reader.CommitMessages(ctx, msg); err != nil {
						c.log.Error("offset commit failed",
							zap.Error(err),
							zap.Int64("offset", msg.Offset),
						)
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

func (c *Consumer) processMessage(ctx context.Context, msg kafkago.Message) {
	traceID := headerValue(msg.Headers, "x-trace-id")
	idempotencyKey := headerValue(msg.Headers, "x-idempotency-key")

	ctx, span := consumerTracer.Start(ctx, "consumer.processMessage")
	defer span.End()
	span.SetAttributes(attribute.String("trace.id", traceID))

	if len(msg.Value) == 0 {
		c.log.Warn("empty kafka message — discarding")
		return
	}

	var blueprint models.AllocationBlueprint
	if err := json.Unmarshal(msg.Value, &blueprint); err != nil {
		c.log.Error("deserialisation failed — routing to DLQ",
			zap.Error(err), zap.String("trace_id", traceID),
		)
		c.producer.PublishDLQ(ctx, msg.Value, "deserialisation_error")
		return
	}

	// Idempotency guard
	if c.redis.IsAlreadyExecuted(ctx, blueprint.BlueprintID) {
		c.log.Info("idempotency hit — skipping",
			zap.String("blueprint_id", blueprint.BlueprintID),
		)
		return
	}

	// Execute blueprint through state machine
	execEvent, rollbackEvent := c.engine.Execute(ctx, &blueprint)

	// Publish execution event
	if err := c.producer.Publish(ctx, execEvent, blueprint.BlueprintID, traceID, idempotencyKey); err != nil {
		c.log.Error("execution event publish failed — suppressing commit",
			zap.Error(err), zap.String("blueprint_id", blueprint.BlueprintID),
		)
		// Do not commit offset — will be replayed
		return
	}

	// Mark as executed to prevent replay
	if err := c.redis.MarkExecuted(ctx, blueprint.BlueprintID); err != nil {
		c.log.Warn("mark executed failed — idempotency risk on replay",
			zap.Error(err), zap.String("blueprint_id", blueprint.BlueprintID),
		)
	}

	// Publish rollback event if execution partially failed
	if rollbackEvent != nil {
		if err := c.producer.Publish(ctx, rollbackEvent, blueprint.BlueprintID, traceID, idempotencyKey); err != nil {
			c.log.Error("rollback event publish failed", zap.Error(err))
		}
	}
}

func headerValue(headers []kafkago.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

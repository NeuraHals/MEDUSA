package kafka

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/antigravity/mono/services/recovery-agent/internal/config"
	"github.com/antigravity/mono/services/recovery-agent/internal/health"
	"github.com/antigravity/mono/services/recovery-agent/internal/models"
	"github.com/antigravity/mono/services/recovery-agent/internal/recovery"
	"github.com/antigravity/mono/services/recovery-agent/internal/state"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer drives the rollback worker pool.
type Consumer struct {
	cfg      *config.Config
	log      *zap.Logger
	workflow *recovery.Workflow
	producer *Producer
	redis    *state.RedisClient
	health   *health.Health
	reader   *kafkago.Reader
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewConsumer(
	cfg *config.Config,
	log *zap.Logger,
	workflow *recovery.Workflow,
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
		CommitInterval: 0,
	})
	return &Consumer{cfg: cfg, log: log, workflow: workflow, producer: producer, redis: redis, health: hc, reader: reader, stopCh: make(chan struct{})}
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
		wid := i
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.log.Info("recovery worker started", zap.Int("worker_id", wid))
			for {
				select {
				case <-c.stopCh:
					return
				case msg := <-msgCh:
					ok := c.processMessage(ctx, msg)
					if ok {
						if err := c.reader.CommitMessages(ctx, msg); err != nil {
							c.log.Error("offset commit failed", zap.Error(err))
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

func (c *Consumer) processMessage(ctx context.Context, msg kafkago.Message) bool {
	traceID := headerValue(msg.Headers, "x-trace-id")
	idempotencyKey := headerValue(msg.Headers, "x-idempotency-key")

	if len(msg.Value) == 0 {
		return true
	}

	var manifest models.RollbackManifest
	if err := json.Unmarshal(msg.Value, &manifest); err != nil {
		c.log.Error("deserialise failed", zap.Error(err))
		c.producer.PublishDLQ(ctx, msg.Value, "deserialisation_error")
		return true
	}

	if manifest.TraceID == "" { manifest.TraceID = traceID }
	if manifest.IdempotencyKey == "" { manifest.IdempotencyKey = idempotencyKey }

	// Idempotency guard
	if c.redis.IsAlreadyRolledBack(ctx, manifest.RollbackID) {
		c.log.Info("idempotency hit — rollback already processed",
			zap.String("rollback_id", manifest.RollbackID),
		)
		return true
	}

	// Execute rollback workflow
	event := c.workflow.Execute(ctx, &manifest)

	// Publish recovery event
	if err := c.producer.PublishRecovery(ctx, event, manifest.CrisisID, manifest.TraceID, manifest.IdempotencyKey); err != nil {
		c.log.Error("recovery event publish failed — suppressing commit",
			zap.Error(err), zap.String("rollback_id", manifest.RollbackID),
		)
		return false // retry
	}

	// Mark as processed after successful publish
	_ = c.redis.MarkRolledBack(ctx, manifest.RollbackID)
	return true
}

func headerValue(headers []kafkago.Header, key string) string {
	for _, h := range headers {
		if h.Key == key { return string(h.Value) }
	}
	return ""
}

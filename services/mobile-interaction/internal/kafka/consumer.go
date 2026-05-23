package kafka

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/antigravity/mono/services/mobile-interaction/internal/config"
	"github.com/antigravity/mono/services/mobile-interaction/internal/health"
	"github.com/antigravity/mono/services/mobile-interaction/internal/mobile"
	"github.com/antigravity/mono/services/mobile-interaction/internal/models"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer struct {
	cfg        *config.Config
	log        *zap.Logger
	reconciler *mobile.Reconciler
	producer   *Producer
	health     *health.Health
	reader     *kafkago.Reader
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func NewConsumer(
	cfg *config.Config,
	log *zap.Logger,
	reconciler *mobile.Reconciler,
	producer *Producer,
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
	return &Consumer{cfg: cfg, log: log, reconciler: reconciler, producer: producer, health: hc, reader: reader, stopCh: make(chan struct{})}
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
			c.log.Info("mobile worker started", zap.Int("worker_id", wid))
			for {
				select {
				case <-c.stopCh:
					return
				case msg := <-msgCh:
					ok := c.processMessage(ctx, msg)
					if ok {
						_ = c.reader.CommitMessages(ctx, msg)
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
	var req models.PushNotificationRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		c.log.Error("deserialise failed", zap.Error(err))
		c.producer.PublishDLQ(ctx, msg.Value, "deserialisation_error")
		return true
	}
	if req.TraceID == "" { req.TraceID = traceID }
	if req.IdempotencyKey == "" { req.IdempotencyKey = idempotencyKey }
	if err := c.reconciler.SendApprovalPrompt(ctx, &req); err != nil {
		c.log.Error("push failed", zap.Error(err))
		c.producer.PublishDLQ(ctx, msg.Value, "push_delivery_error")
	}
	return true
}

func headerValue(headers []kafkago.Header, key string) string {
	for _, h := range headers {
		if h.Key == key { return string(h.Value) }
	}
	return ""
}

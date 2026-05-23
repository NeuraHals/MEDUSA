package kafka

import (
	"context"
	"sync"
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/audit"
	"github.com/antigravity/mono/services/audit-agent/internal/config"
	"github.com/antigravity/mono/services/audit-agent/internal/health"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer subscribes to all orchestration event topics and drives the audit pipeline.
type Consumer struct {
	cfg      *config.Config
	log      *zap.Logger
	pipeline *audit.Pipeline
	producer *Producer
	health   *health.Health
	readers  []*kafkago.Reader
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewConsumer(
	cfg *config.Config,
	log *zap.Logger,
	pipeline *audit.Pipeline,
	producer *Producer,
	hc *health.Health,
) *Consumer {
	readers := make([]*kafkago.Reader, 0, len(cfg.KafkaInputTopics))
	for _, topic := range cfg.KafkaInputTopics {
		readers = append(readers, kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:        cfg.KafkaBrokers,
			Topic:          topic,
			GroupID:        cfg.KafkaConsumerGroup,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: 0,
		}))
	}
	return &Consumer{
		cfg:      cfg,
		log:      log,
		pipeline: pipeline,
		producer: producer,
		health:   hc,
		readers:  readers,
		stopCh:   make(chan struct{}),
	}
}

// Start launches one reader goroutine per topic and a shared bounded worker pool.
func (c *Consumer) Start(ctx context.Context) {
	msgCh := make(chan kafkago.Message, 512)

	for _, reader := range c.readers {
		r := reader
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			for {
				msg, err := r.ReadMessage(ctx)
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
	}

	for i := 0; i < c.cfg.KafkaWorkerCount; i++ {
		wid := i
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.log.Info("audit worker started", zap.Int("worker_id", wid))
			for {
				select {
				case <-c.stopCh:
					return
				case msg := <-msgCh:
					committed := c.processMessage(ctx, msg)
					if committed {
						// Find the reader for this topic and commit
						for _, r := range c.readers {
							if r.Config().Topic == msg.Topic {
								if err := r.CommitMessages(ctx, msg); err != nil {
									c.log.Error("offset commit failed", zap.Error(err))
								}
								break
							}
						}
					}
				}
			}
		}()
	}
}

func (c *Consumer) Stop() {
	close(c.stopCh)
	for _, r := range c.readers {
		_ = r.Close()
	}
	c.wg.Wait()
}

func (c *Consumer) processMessage(ctx context.Context, msg kafkago.Message) bool {
	traceID := headerValue(msg.Headers, "x-trace-id")
	agentID := headerValue(msg.Headers, "x-producer-agent")

	if len(msg.Value) == 0 {
		c.log.Warn("empty kafka message — discarding")
		return true
	}

	if err := c.pipeline.Process(ctx, msg.Value, traceID, agentID); err != nil {
		c.log.Error("audit pipeline failed — routing to DLQ",
			zap.Error(err),
			zap.String("topic", msg.Topic),
			zap.Int64("offset", msg.Offset),
		)
		c.producer.PublishDLQ(ctx, msg.Value, "audit_pipeline_error")
		return true // DLQ is terminal — commit the offset
	}

	return true
}

func headerValue(headers []kafkago.Header, key string) string {
	for _, h := range headers {
		if h.Key == key { return string(h.Value) }
	}
	return ""
}

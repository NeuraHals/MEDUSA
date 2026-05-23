package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/mono/services/recovery-agent/internal/config"
	"github.com/antigravity/mono/services/recovery-agent/internal/health"
	agenthttp "github.com/antigravity/mono/services/recovery-agent/internal/http"
	"github.com/antigravity/mono/services/recovery-agent/internal/kafka"
	"github.com/antigravity/mono/services/recovery-agent/internal/logger"
	"github.com/antigravity/mono/services/recovery-agent/internal/otel"
	"github.com/antigravity/mono/services/recovery-agent/internal/recovery"
	"github.com/antigravity/mono/services/recovery-agent/internal/rollback"
	"github.com/antigravity/mono/services/recovery-agent/internal/state"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	shutdown, err := otel.InitTracer(context.Background(), cfg.OTelEndpoint, "recovery-agent")
	if err != nil {
		log.Fatal("otel init failed", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	}()

	redis, err := state.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal("redis init failed", zap.Error(err))
	}

	executor := rollback.NewExecutor(log, cfg.UndoTimeoutSecs)
	workflow := recovery.NewWorkflow(
		log, redis, executor,
		cfg.MaxRetries, cfg.DegradedMode, cfg.SchemaVersion,
		cfg.CBFailureThreshold, cfg.CBRecoverySecs,
	)

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaOutputTopic, cfg.KafkaDLQTopic, cfg.SchemaVersion)
	defer producer.Close()

	hc := health.New()

	consumer := kafka.NewConsumer(cfg, log, workflow, producer, redis, hc)
	go consumer.Start(context.Background())

	app := agenthttp.NewRouter(log, hc, cfg, redis)
	go func() {
		log.Info("recovery-agent HTTP listening", zap.String("addr", cfg.HTTPAddr))
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Fatal("fiber error", zap.Error(err))
		}
	}()

	hc.SetReady(true)
	log.Info("recovery-agent ready",
		zap.Bool("degraded_mode", cfg.DegradedMode),
		zap.Int("max_retries", cfg.MaxRetries),
		zap.Int("undo_timeout_secs", cfg.UndoTimeoutSecs),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")
	hc.SetReady(false)
	consumer.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)
	log.Info("recovery-agent stopped")
}

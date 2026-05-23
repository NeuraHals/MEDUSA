package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/audit"
	"github.com/antigravity/mono/services/audit-agent/internal/config"
	"github.com/antigravity/mono/services/audit-agent/internal/health"
	agenthttp "github.com/antigravity/mono/services/audit-agent/internal/http"
	"github.com/antigravity/mono/services/audit-agent/internal/kafka"
	"github.com/antigravity/mono/services/audit-agent/internal/logger"
	"github.com/antigravity/mono/services/audit-agent/internal/otel"
	"github.com/antigravity/mono/services/audit-agent/internal/replay"
	"github.com/antigravity/mono/services/audit-agent/internal/retention"
	"github.com/antigravity/mono/services/audit-agent/internal/state"
	"github.com/antigravity/mono/services/audit-agent/internal/storage"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	shutdown, err := otel.InitTracer(context.Background(), cfg.OTelEndpoint, "audit-agent")
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

	worm, err := storage.NewWORMClient(
		context.Background(), log,
		cfg.S3Region, cfg.S3Bucket, cfg.S3Prefix, cfg.S3ObjectLockMode,
	)
	if err != nil {
		log.Fatal("WORM client init failed", zap.Error(err))
	}

	policy := &retention.Policy{
		StandardDays:  cfg.RetentionDays,
		ExtendedDays:  cfg.ExtendedRetentionDays,
		ForensicDays:  cfg.ForensicRetentionDays,
	}

	indexer := replay.NewIndexer(log, redis)
	pipeline := audit.NewPipeline(log, redis, worm, indexer, policy, cfg.SchemaVersion)

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaDLQTopic)
	hc := health.New()

	consumer := kafka.NewConsumer(cfg, log, pipeline, producer, hc)
	go consumer.Start(context.Background())

	app := agenthttp.NewRouter(log, hc, cfg, indexer)
	go func() {
		log.Info("audit-agent HTTP listening", zap.String("addr", cfg.HTTPAddr))
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Fatal("fiber error", zap.Error(err))
		}
	}()

	hc.SetReady(true)
	log.Info("audit-agent ready",
		zap.Strings("topics", cfg.KafkaInputTopics),
		zap.String("s3_bucket", cfg.S3Bucket),
		zap.String("object_lock_mode", cfg.S3ObjectLockMode),
		zap.Int("retention_days", cfg.RetentionDays),
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
	log.Info("audit-agent stopped")
}

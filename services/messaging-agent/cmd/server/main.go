package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/mono/services/messaging-agent/internal/config"
	"github.com/antigravity/mono/services/messaging-agent/internal/health"
	agenthttp "github.com/antigravity/mono/services/messaging-agent/internal/http"
	"github.com/antigravity/mono/services/messaging-agent/internal/kafka"
	"github.com/antigravity/mono/services/messaging-agent/internal/logger"
	"github.com/antigravity/mono/services/messaging-agent/internal/messaging"
	"github.com/antigravity/mono/services/messaging-agent/internal/otel"
	"github.com/antigravity/mono/services/messaging-agent/internal/providers"
	"github.com/antigravity/mono/services/messaging-agent/internal/state"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	shutdown, err := otel.InitTracer(context.Background(), cfg.OTelEndpoint, "messaging-agent")
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

	pd := providers.NewPagerDutyProvider(cfg.PagerDutyAPIKey, cfg.PagerDutyBaseURL)
	twilio := providers.NewTwilioProvider(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber, cfg.TwilioBaseURL)
	push := providers.NewPushProvider(cfg.APNsKeyPath, cfg.APNsTeamID, cfg.APNsBundleID, cfg.FCMServerKey)

	dispatcher := messaging.NewDispatcher(
		log, redis, pd, twilio, push,
		cfg.MaxRetries, cfg.DegradedMode,
		cfg.CBFailureThreshold, cfg.CBRecoverySeconds,
	)

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaOutputTopic, cfg.KafkaDLQTopic, cfg.SchemaVersion)
	defer producer.Close()

	hc := health.New()

	consumer := kafka.NewConsumer(cfg, log, dispatcher, producer, hc)
	go consumer.Start(context.Background())

	httpApp := agenthttp.NewRouter(log, hc, cfg)
	go func() {
		log.Info("messaging-agent HTTP listening", zap.String("addr", cfg.HTTPAddr))
		if err := httpApp.Listen(cfg.HTTPAddr); err != nil {
			log.Fatal("fiber listen error", zap.Error(err))
		}
	}()

	hc.SetReady(true)
	log.Info("messaging-agent ready",
		zap.Bool("degraded_mode", cfg.DegradedMode),
		zap.Int("worker_count", cfg.KafkaWorkerCount),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully")
	hc.SetReady(false)
	consumer.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpApp.ShutdownWithContext(ctx)
	log.Info("messaging-agent stopped")
}

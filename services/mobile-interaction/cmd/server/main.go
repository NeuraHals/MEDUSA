package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/mono/services/mobile-interaction/internal/cache"
	"github.com/antigravity/mono/services/mobile-interaction/internal/config"
	agentgrpc "github.com/antigravity/mono/services/mobile-interaction/internal/grpc"
	"github.com/antigravity/mono/services/mobile-interaction/internal/health"
	agenthttp "github.com/antigravity/mono/services/mobile-interaction/internal/http"
	"github.com/antigravity/mono/services/mobile-interaction/internal/kafka"
	"github.com/antigravity/mono/services/mobile-interaction/internal/logger"
	"github.com/antigravity/mono/services/mobile-interaction/internal/mobile"
	"github.com/antigravity/mono/services/mobile-interaction/internal/otel"
	"github.com/antigravity/mono/services/mobile-interaction/internal/state"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	shutdown, err := otel.InitTracer(context.Background(), cfg.OTelEndpoint, "mobile-interaction")
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

	hoaClient, err := agentgrpc.NewHOAClient(log, cfg.HOAGRPCEndpoint)
	if err != nil {
		log.Fatal("HOA gRPC client init failed", zap.Error(err))
	}
	defer hoaClient.Close()

	pushClient := mobile.NewPushClient(log, cfg.APNsKeyPath, cfg.APNsTeamID, cfg.APNsBundleID, cfg.FCMServerKey)
	smsClient := mobile.NewSMSClient(log, cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber, cfg.TwilioBaseURL)

	reconciler := mobile.NewReconciler(
		log, redis, pushClient, smsClient,
		cfg.OfflineQueueTTLSecs, cfg.SessionTTLSecs, cfg.DegradedMode,
	)

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaOutputTopic, cfg.KafkaDLQTopic, cfg.SchemaVersion)
	defer producer.Close()

	memCache := cache.New()
	hc := health.New()

	consumer := kafka.NewConsumer(cfg, log, reconciler, producer, hc)
	go consumer.Start(context.Background())

	app := agenthttp.NewRouter(log, hc, cfg, redis, reconciler, hoaClient, producer, memCache)
	go func() {
		log.Info("mobile-interaction HTTP listening", zap.String("addr", cfg.HTTPAddr))
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Fatal("fiber error", zap.Error(err))
		}
	}()

	hc.SetReady(true)
	log.Info("mobile-interaction ready",
		zap.Bool("degraded_mode", cfg.DegradedMode),
		zap.Int("offline_queue_ttl_secs", cfg.OfflineQueueTTLSecs),
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
	log.Info("mobile-interaction stopped")
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/mono/services/orchestrator-agent/internal/config"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/grpc"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/health"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/http"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/kafka"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/logger"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/otel"
	"github.com/antigravity/mono/services/orchestrator-agent/internal/state"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	// OpenTelemetry
	shutdown, err := otel.InitTracer(context.Background(), cfg.OTelEndpoint, "orchestrator-agent")
	if err != nil {
		log.Fatal("otel init failed", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	}()

	// Redis idempotency store
	redisClient, err := state.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal("redis init failed", zap.Error(err))
	}

	// Kafka output producer (execution events + DLQ)
	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaOutputTopic)
	defer producer.Close()

	// Health tracker
	hc := health.New()

	// Kafka consumer + execution worker pool
	consumer := kafka.NewConsumer(cfg, log, producer, redisClient, hc)
	go consumer.Start(context.Background())

	// HTTP health endpoints
	httpApp := http.NewRouter(log, hc, cfg)
	go func() {
		log.Info("orchestrator-agent HTTP listening", zap.String("addr", cfg.HTTPAddr))
		if err := httpApp.Listen(cfg.HTTPAddr); err != nil {
			log.Fatal("fiber listen error", zap.Error(err))
		}
	}()

	// gRPC ExecutionService
	grpcServer := grpc.NewExecutionServer(log, redisClient)
	go func() {
		log.Info("orchestrator-agent gRPC listening", zap.String("addr", cfg.GRPCAddr))
		if err := grpcServer.Serve(cfg.GRPCAddr); err != nil {
			log.Fatal("grpc serve error", zap.Error(err))
		}
	}()

	hc.SetReady(true)
	log.Info("orchestrator-agent ready")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully")
	hc.SetReady(false)
	consumer.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpApp.ShutdownWithContext(ctx)
	grpcServer.GracefulStop()
	log.Info("orchestrator-agent stopped")
}

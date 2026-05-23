package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/approval"
	"github.com/antigravity/mono/services/oversight-agent/internal/config"
	"github.com/antigravity/mono/services/oversight-agent/internal/grpc"
	"github.com/antigravity/mono/services/oversight-agent/internal/health"
	agenthttp "github.com/antigravity/mono/services/oversight-agent/internal/http"
	"github.com/antigravity/mono/services/oversight-agent/internal/kafka"
	"github.com/antigravity/mono/services/oversight-agent/internal/logger"
	"github.com/antigravity/mono/services/oversight-agent/internal/otel"
	"github.com/antigravity/mono/services/oversight-agent/internal/state"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	shutdown, err := otel.InitTracer(context.Background(), cfg.OTelEndpoint, "oversight-agent")
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

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaOutputTopic, cfg.KafkaDLQTopic, cfg.SchemaVersion)
	defer producer.Close()

	svc := approval.NewService(log, redis, producer, cfg.ApprovalTimeoutSecs, cfg.SchemaVersion)
	defer svc.Stop()

	hc := health.New()

	httpApp := agenthttp.NewRouter(log, hc, cfg, svc)
	go func() {
		log.Info("oversight-agent HTTP listening", zap.String("addr", cfg.HTTPAddr))
		if err := httpApp.Listen(cfg.HTTPAddr); err != nil {
			log.Fatal("fiber listen error", zap.Error(err))
		}
	}()

	grpcServer := grpc.NewApprovalServer(log, svc)
	go func() {
		log.Info("oversight-agent gRPC listening", zap.String("addr", cfg.GRPCAddr))
		if err := grpcServer.Serve(cfg.GRPCAddr); err != nil {
			log.Fatal("grpc serve error", zap.Error(err))
		}
	}()

	hc.SetReady(true)
	log.Info("oversight-agent ready",
		zap.Int("glass_break_timeout_secs", cfg.ApprovalTimeoutSecs),
		zap.Bool("degraded_mode", cfg.DegradedMode),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully")
	hc.SetReady(false)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpApp.ShutdownWithContext(ctx)
	grpcServer.GracefulStop()
	log.Info("oversight-agent stopped")
}

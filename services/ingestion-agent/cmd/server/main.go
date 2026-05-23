package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antigravity/mono/services/ingestion-agent/internal/config"
	"github.com/antigravity/mono/services/ingestion-agent/internal/health"
	agenthttp "github.com/antigravity/mono/services/ingestion-agent/internal/http"
	"github.com/antigravity/mono/services/ingestion-agent/internal/kafka"
	"github.com/antigravity/mono/services/ingestion-agent/internal/logger"
	"github.com/antigravity/mono/services/ingestion-agent/internal/otel"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	defer log.Sync() //nolint:errcheck

	// --- OpenTelemetry ---
	shutdown, err := otel.InitTracer(context.Background(), cfg.OTelEndpoint, "ingestion-agent")
	if err != nil {
		log.Fatal("failed to init tracer", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			log.Error("tracer shutdown error", zap.Error(err))
		}
	}()

	// --- Kafka Producer ---
	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer producer.Close()

	// --- Health ---
	hc := health.New()

	// --- HTTP Router ---
	app := agenthttp.NewRouter(log, producer, hc, cfg)

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("ingestion-agent listening", zap.String("addr", cfg.HTTPAddr))
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			log.Fatal("fiber listen error", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error("fiber shutdown error", zap.Error(err))
	}

	log.Info("ingestion-agent stopped")
}

package http

import (
	"github.com/antigravity/mono/services/ingestion-agent/internal/config"
	"github.com/antigravity/mono/services/ingestion-agent/internal/health"
	"github.com/antigravity/mono/services/ingestion-agent/internal/kafka"
	"github.com/antigravity/mono/services/ingestion-agent/internal/middleware"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
)

// NewRouter assembles the Fiber application with all routes and middleware.
func NewRouter(log *zap.Logger, producer *kafka.Producer, hc *health.Health, cfg *config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// --- Global middleware ---
	app.Use(middleware.Recovery(log))
	app.Use(middleware.RequestID())
	app.Use(otelfiber.Middleware())
	app.Use(fiberlogger.New())
	app.Use(compress.New())

	// --- Health endpoints ---
	app.Get("/health/live", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if !hc.IsReady() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// --- API routes ---
	handler := NewWebhookHandler(log, producer, cfg.SchemaVersion)
	v1 := app.Group("/v1")
	v1.Post("/telemetry", handler.Handle)

	hc.SetReady(true)
	return app
}

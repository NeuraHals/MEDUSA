package http

import (
	"github.com/antigravity/mono/services/oversight-agent/internal/approval"
	"github.com/antigravity/mono/services/oversight-agent/internal/config"
	"github.com/antigravity/mono/services/oversight-agent/internal/health"
	"github.com/antigravity/mono/services/oversight-agent/internal/middleware"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
)

func NewRouter(
	log *zap.Logger,
	hc *health.Health,
	cfg *config.Config,
	svc *approval.Service,
) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.Recovery(log))
	app.Use(middleware.RequestID())
	app.Use(otelfiber.Middleware())
	app.Use(fiberlogger.New())
	app.Use(compress.New())

	app.Get("/health/live", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if !hc.IsReady() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// Operator API — manual approval/denial (used by SOC dashboard)
	h := NewHandlers(log, svc)
	v1 := app.Group("/v1")
	v1.Post("/approval/:blueprint_id/approve", h.Approve)
	v1.Post("/approval/:blueprint_id/deny", h.Deny)
	v1.Get("/approval/:blueprint_id", h.GetStatus)

	return app
}

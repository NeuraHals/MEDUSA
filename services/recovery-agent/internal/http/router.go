package http

import (
	"github.com/antigravity/mono/services/recovery-agent/internal/config"
	"github.com/antigravity/mono/services/recovery-agent/internal/health"
	"github.com/antigravity/mono/services/recovery-agent/internal/middleware"
	"github.com/antigravity/mono/services/recovery-agent/internal/state"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
)

func NewRouter(log *zap.Logger, hc *health.Health, cfg *config.Config, redis *state.RedisClient) *fiber.App {
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

	app.Get("/health/live", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if !hc.IsReady() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	v1 := app.Group("/v1")

	// SOC operator query: check rollback execution state
	v1.Get("/rollback/:rollback_id", func(c *fiber.Ctx) error {
		rollbackID := c.Params("rollback_id")
		if rollbackID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "rollback_id required"})
		}
		record, err := redis.GetRecord(c.UserContext(), rollbackID)
		if err != nil {
			log.Error("get record failed", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if record == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.JSON(record)
	})

	return app
}

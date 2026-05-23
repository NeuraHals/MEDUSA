package http

import (
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/config"
	"github.com/antigravity/mono/services/audit-agent/internal/health"
	"github.com/antigravity/mono/services/audit-agent/internal/middleware"
	"github.com/antigravity/mono/services/audit-agent/internal/replay"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
)

func NewRouter(
	log *zap.Logger,
	hc *health.Health,
	cfg *config.Config,
	indexer *replay.Indexer,
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

	app.Get("/health/live", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if !hc.IsReady() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	v1 := app.Group("/v1")

	// Forensic replay query — returns the ordered audit chain for a crisis
	v1.Get("/audit/replay/:crisis_id", func(c *fiber.Ctx) error {
		crisisID := c.Params("crisis_id")
		if crisisID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "crisis_id required"})
		}
		fromStr := c.Query("from", time.Now().AddDate(0, 0, -7).Format(time.RFC3339))
		toStr := c.Query("to", time.Now().Format(time.RFC3339))

		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid from timestamp"})
		}
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid to timestamp"})
		}

		entries, err := indexer.Query(c.UserContext(), crisisID, from, to)
		if err != nil {
			log.Error("replay query failed", zap.Error(err), zap.String("crisis_id", crisisID))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"crisis_id": crisisID,
			"count":     len(entries),
			"entries":   entries,
		})
	})

	return app
}

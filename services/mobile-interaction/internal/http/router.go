package http

import (
	"time"

	agentgrpc "github.com/antigravity/mono/services/mobile-interaction/internal/grpc"
	"github.com/antigravity/mono/services/mobile-interaction/internal/cache"
	"github.com/antigravity/mono/services/mobile-interaction/internal/config"
	"github.com/antigravity/mono/services/mobile-interaction/internal/health"
	"github.com/antigravity/mono/services/mobile-interaction/internal/kafka"
	"github.com/antigravity/mono/services/mobile-interaction/internal/middleware"
	"github.com/antigravity/mono/services/mobile-interaction/internal/mobile"
	"github.com/antigravity/mono/services/mobile-interaction/internal/models"
	"github.com/antigravity/mono/services/mobile-interaction/internal/security"
	"github.com/antigravity/mono/services/mobile-interaction/internal/state"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func NewRouter(
	log *zap.Logger,
	hc *health.Health,
	cfg *config.Config,
	redis *state.RedisClient,
	reconciler *mobile.Reconciler,
	hoaClient *agentgrpc.HOAClient,
	producer *kafka.Producer,
	memCache *cache.MemoryCache,
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

	// Health
	app.Get("/health/live", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/health/ready", func(c *fiber.Ctx) error {
		if !hc.IsReady() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	v1 := app.Group("/v1")

	// Device registration — called when app launches or re-authenticates
	v1.Post("/device/register", func(c *fiber.Ctx) error {
		var body struct {
			OperatorID  string `json:"operator_id"`
			DeviceToken string `json:"device_token"`
			Platform    string `json:"platform"`
			Phone       string `json:"phone"`
		}
		if err := c.BodyParser(&body); err != nil || body.OperatorID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
		}
		if err := reconciler.RegisterDevice(c.UserContext(), body.OperatorID, body.DeviceToken, body.Platform, body.Phone); err != nil {
			log.Error("register device failed", zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"registered": true, "operator_id": body.OperatorID})
	})

	// Biometric approval submission — called by mobile app after FaceID/fingerprint
	v1.Post("/approval/submit", func(c *fiber.Ctx) error {
		var req models.ApprovalRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
		}
		if req.BlueprintID == "" || req.OperatorID == "" || req.Decision == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "blueprint_id, operator_id, decision required"})
		}

		// Idempotency guard
		if redis.IsApprovalDuplicate(c.UserContext(), req.IdempotencyKey) {
			return c.JSON(fiber.Map{"accepted": true, "duplicate": true})
		}

		// Validate biometric JWT
		_, valid := security.ValidateBiometricJWT(req.BiometricJWT)
		if !valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid biometric token"})
		}

		// Publish relay event to Kafka
		relayEvent := &models.ApprovalRelayEvent{
			EventID:        uuid.New().String(),
			BlueprintID:    req.BlueprintID,
			ActionID:       req.ActionID,
			HospitalID:     req.HospitalID,
			Decision:       req.Decision,
			OperatorID:     req.OperatorID,
			BiometricJWT:   security.HashToken(req.BiometricJWT), // never relay raw JWT
			TraceID:        req.TraceID,
			IdempotencyKey: req.IdempotencyKey,
			SchemaVersion:  cfg.SchemaVersion,
		}

		if err := producer.PublishRelay(c.UserContext(), relayEvent, req.BlueprintID, req.TraceID, req.IdempotencyKey); err != nil {
			log.Error("relay publish failed", zap.Error(err), zap.String("blueprint_id", req.BlueprintID))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "publish failed"})
		}

		_ = redis.MarkApprovalProcessed(c.UserContext(), req.IdempotencyKey)
		return c.JSON(fiber.Map{"accepted": true, "blueprint_id": req.BlueprintID, "decision": req.Decision})
	})

	// Operator heartbeat — updates online status and drains offline queue
	v1.Post("/operator/heartbeat", func(c *fiber.Ctx) error {
		var body struct {
			OperatorID string `json:"operator_id"`
		}
		if err := c.BodyParser(&body); err != nil || body.OperatorID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "operator_id required"})
		}
		if err := redis.MarkOperatorOnline(c.UserContext(), body.OperatorID, true); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"online": true, "operator_id": body.OperatorID, "at": time.Now().UTC()})
	})

	return app
}

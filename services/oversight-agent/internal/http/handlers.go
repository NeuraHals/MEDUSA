package http

import (
	"github.com/antigravity/mono/services/oversight-agent/internal/approval"
	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/antigravity/mono/services/oversight-agent/internal/security"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handlers struct {
	log *zap.Logger
	svc *approval.Service
}

func NewHandlers(log *zap.Logger, svc *approval.Service) *Handlers {
	return &Handlers{log: log, svc: svc}
}

// Approve handles POST /v1/approval/:blueprint_id/approve
// Used by SOC operators via the MIA mobile dashboard.
func (h *Handlers) Approve(c *fiber.Ctx) error {
	blueprintID := c.Params("blueprint_id")
	if blueprintID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "blueprint_id required"})
	}

	var body struct {
		ApproverID   string `json:"approver_id"`
		BiometricJWT string `json:"biometric_jwt"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if !security.SanitiseApproverID(body.ApproverID) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid approver_id"})
	}

	record, err := h.svc.GetStore().Resolve(c.UserContext(), blueprintID, models.StatusApproved, body.ApproverID, body.BiometricJWT)
	if err != nil {
		h.log.Error("approve failed", zap.Error(err), zap.String("blueprint_id", blueprintID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"blueprint_id": blueprintID, "status": record.Status})
}

// Deny handles POST /v1/approval/:blueprint_id/deny
func (h *Handlers) Deny(c *fiber.Ctx) error {
	blueprintID := c.Params("blueprint_id")
	if blueprintID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "blueprint_id required"})
	}

	var body struct {
		ApproverID string `json:"approver_id"`
		Reason     string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	record, err := h.svc.GetStore().Resolve(c.UserContext(), blueprintID, models.StatusDenied, body.ApproverID, "")
	if err != nil {
		h.log.Error("deny failed", zap.Error(err), zap.String("blueprint_id", blueprintID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"blueprint_id": blueprintID, "status": record.Status})
}

// GetStatus handles GET /v1/approval/:blueprint_id
func (h *Handlers) GetStatus(c *fiber.Ctx) error {
	blueprintID := c.Params("blueprint_id")
	record, err := h.svc.GetStore().Get(c.UserContext(), blueprintID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if record == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(record)
}

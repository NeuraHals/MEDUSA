package http

import (
	"encoding/json"
	"time"

	"github.com/antigravity/mono/services/ingestion-agent/internal/kafka"
	"github.com/antigravity/mono/services/ingestion-agent/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("ingestion-agent/webhook")

// WebhookHandler handles POST /v1/telemetry.
type WebhookHandler struct {
	log      *zap.Logger
	producer *kafka.Producer
	version  string
}

// NewWebhookHandler constructs the handler.
func NewWebhookHandler(log *zap.Logger, producer *kafka.Producer, schemaVersion string) *WebhookHandler {
	return &WebhookHandler{log: log, producer: producer, version: schemaVersion}
}

// Handle validates the incoming telemetry payload, normalises it into a
// UnifiedEvent, and publishes it to the Kafka ingestion topic.
func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	ctx, span := tracer.Start(c.UserContext(), "webhook.handle")
	defer span.End()

	var req models.TelemetryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid JSON body",
		})
	}

	// --- Required field validation ---
	if req.SourceSystem == "" || req.EventType == "" || req.HospitalID == "" || req.TimestampUTC == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "missing required fields: source_system, event_type, hospital_id, timestamp_utc",
		})
	}

	ts, err := time.Parse(time.RFC3339, req.TimestampUTC)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "timestamp_utc must be RFC3339",
		})
	}

	// --- Event enrichment ---
	eventID := uuid.New().String()
	traceID := span.SpanContext().TraceID().String()

	span.SetAttributes(
		attribute.String("event.id", eventID),
		attribute.String("hospital.id", req.HospitalID),
		attribute.String("source.system", req.SourceSystem),
	)

	event := models.UnifiedEvent{
		EventID:        eventID,
		SourceSystem:   req.SourceSystem,
		EventType:      req.EventType,
		HospitalID:     req.HospitalID,
		TimestampUTC:   ts,
		Payload:        req.Payload,
		TraceID:        traceID,
		IdempotencyKey: uuid.New().String(),
		SchemaVersion:  h.version,
		IngestedAt:     time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		h.log.Error("failed to marshal event", zap.Error(err), zap.String("event_id", eventID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "serialisation failure"})
	}

	// --- Kafka publish ---
	if err := h.producer.Publish(ctx, payload, eventID); err != nil {
		h.log.Error("kafka publish failed",
			zap.Error(err),
			zap.String("event_id", eventID),
			zap.String("trace_id", traceID),
		)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "event bus unavailable"})
	}

	h.log.Info("event ingested",
		zap.String("event_id", eventID),
		zap.String("hospital_id", req.HospitalID),
		zap.String("source_system", req.SourceSystem),
		zap.String("trace_id", traceID),
	)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"event_id": eventID,
		"trace_id": traceID,
		"status":   "accepted",
	})
}

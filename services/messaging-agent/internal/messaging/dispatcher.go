package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/messaging-agent/internal/middleware"
	"github.com/antigravity/mono/services/messaging-agent/internal/models"
	"github.com/antigravity/mono/services/messaging-agent/internal/providers"
	"github.com/antigravity/mono/services/messaging-agent/internal/state"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("messaging-agent/dispatcher")

// Dispatcher routes a NotificationRequest through the correct providers,
// applying circuit-breaking, retries, and SMS fallback for failed channels.
type Dispatcher struct {
	log          *zap.Logger
	redis        *state.RedisClient
	pagerduty    *providers.PagerDutyProvider
	twilio       *providers.TwilioProvider
	push         *providers.PushProvider
	maxRetries   int
	degradedMode bool
	// Per-provider circuit breakers
	cbPagerDuty *middleware.CircuitBreaker
	cbTwilio    *middleware.CircuitBreaker
	cbPush      *middleware.CircuitBreaker
}

func NewDispatcher(
	log *zap.Logger,
	redis *state.RedisClient,
	pd *providers.PagerDutyProvider,
	twilio *providers.TwilioProvider,
	push *providers.PushProvider,
	maxRetries int,
	degradedMode bool,
	cbFailureThreshold int,
	cbRecoverySecs int,
) *Dispatcher {
	return &Dispatcher{
		log:          log,
		redis:        redis,
		pagerduty:    pd,
		twilio:       twilio,
		push:         push,
		maxRetries:   maxRetries,
		degradedMode: degradedMode,
		cbPagerDuty:  middleware.NewCircuitBreaker(cbFailureThreshold, cbRecoverySecs),
		cbTwilio:     middleware.NewCircuitBreaker(cbFailureThreshold, cbRecoverySecs),
		cbPush:       middleware.NewCircuitBreaker(cbFailureThreshold, cbRecoverySecs),
	}
}

// Dispatch fans out a notification request to all requested channels and recipients.
// Returns a slice of DeliveryResult, one per recipient per channel.
func (d *Dispatcher) Dispatch(ctx context.Context, req *models.NotificationRequest) []models.DeliveryResult {
	ctx, span := tracer.Start(ctx, "dispatcher.dispatch")
	defer span.End()
	span.SetAttributes(
		attribute.String("request.id", req.RequestID),
		attribute.String("crisis.id", req.CrisisID),
		attribute.Int("channel.count", len(req.Channels)),
		attribute.Int("recipient.count", len(req.Recipients)),
	)

	results := make([]models.DeliveryResult, 0, len(req.Channels)*len(req.Recipients))

	for _, recipient := range req.Recipients {
		// In degraded mode, only SMS is attempted
		channels := req.Channels
		if d.degradedMode {
			channels = []models.Channel{models.ChannelSMS}
			d.log.Warn("degraded mode active — SMS-only fanout",
				zap.String("recipient_id", recipient.RecipientID),
			)
		}

		for _, ch := range channels {
			result := d.deliverWithFallback(ctx, req, recipient, ch)
			results = append(results, result)

			// If primary channel fails, attempt SMS fallback (unless already SMS)
			if !result.Success && ch != models.ChannelSMS {
				d.log.Warn("primary channel failed — attempting SMS fallback",
					zap.String("channel", string(ch)),
					zap.String("recipient_id", recipient.RecipientID),
				)
				fallback := d.deliverChannel(ctx, req, recipient, models.ChannelSMS)
				results = append(results, fallback)
			}
		}
	}

	return results
}

func (d *Dispatcher) deliverWithFallback(
	ctx context.Context,
	req *models.NotificationRequest,
	recipient models.Recipient,
	ch models.Channel,
) models.DeliveryResult {
	result := d.deliverChannel(ctx, req, recipient, ch)
	return result
}

func (d *Dispatcher) deliverChannel(
	ctx context.Context,
	req *models.NotificationRequest,
	recipient models.Recipient,
	ch models.Channel,
) models.DeliveryResult {
	base := models.DeliveryResult{
		RequestID:      req.RequestID,
		RecipientID:    recipient.RecipientID,
		Channel:        ch,
		TraceID:        req.TraceID,
		IdempotencyKey: fmt.Sprintf("%s:%s:%s", req.IdempotencyKey, string(ch), recipient.RecipientID),
		SchemaVersion:  req.SchemaVersion,
		DeliveredAt:    time.Now().UTC(),
	}

	// Idempotency check
	if d.redis.IsAlreadyDelivered(ctx, base.IdempotencyKey) {
		d.log.Info("idempotency hit — delivery skipped",
			zap.String("idempotency_key", base.IdempotencyKey),
		)
		base.Success = true
		base.ProviderRef = "idempotency-hit"
		return base
	}

	var providerFn func() (string, error)
	var cb *middleware.CircuitBreaker

	switch ch {
	case models.ChannelPagerDuty:
		cb = d.cbPagerDuty
		providerFn = func() (string, error) { return d.pagerduty.Deliver(ctx, req, recipient) }
	case models.ChannelSMS:
		cb = d.cbTwilio
		providerFn = func() (string, error) { return d.twilio.Deliver(ctx, req, recipient) }
	case models.ChannelAPNs:
		cb = d.cbPush
		providerFn = func() (string, error) { return d.push.Deliver(ctx, req, recipient) }
	case models.ChannelFCM:
		cb = d.cbPush
		providerFn = func() (string, error) { return d.push.Deliver(ctx, req, recipient) }
	default:
		base.ErrorCode = "UNKNOWN_CHANNEL"
		return base
	}

	if cb.IsOpen() {
		d.log.Warn("circuit breaker open — skipping delivery",
			zap.String("channel", string(ch)),
			zap.String("recipient_id", recipient.RecipientID),
		)
		base.ErrorCode = "CIRCUIT_OPEN"
		return base
	}

	var ref string
	var attempts int
	err := middleware.Retry(ctx, d.log, d.maxRetries, func() error {
		attempts++
		var err error
		ref, err = providerFn()
		return err
	})

	base.Retries = attempts - 1

	if err != nil {
		cb.RecordFailure()
		d.log.Error("delivery failed after retries",
			zap.String("channel", string(ch)),
			zap.String("recipient_id", recipient.RecipientID),
			zap.Error(err),
		)
		base.ErrorCode = err.Error()
		return base
	}

	cb.RecordSuccess()
	base.Success = true
	base.ProviderRef = ref
	_ = d.redis.MarkDelivered(ctx, base.IdempotencyKey)
	return base
}

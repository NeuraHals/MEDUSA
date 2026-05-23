package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/antigravity/mono/services/messaging-agent/internal/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var pdTracer = otel.Tracer("messaging-agent/pagerduty")

// PagerDutyPayload is the Events API v2 payload schema.
type PagerDutyPayload struct {
	RoutingKey  string            `json:"routing_key"`
	EventAction string            `json:"event_action"`
	DedupKey    string            `json:"dedup_key"`
	Payload     PDInnerPayload    `json:"payload"`
	Client      string            `json:"client"`
	Links       []PDLink          `json:"links,omitempty"`
}

type PDInnerPayload struct {
	Summary   string            `json:"summary"`
	Source    string            `json:"source"`
	Severity  string            `json:"severity"` // critical, error, warning, info
	Timestamp string            `json:"timestamp"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}

type PDLink struct {
	HRef string `json:"href"`
	Text string `json:"text"`
}

// PagerDutyProvider delivers incidents to PagerDuty Events API v2.
type PagerDutyProvider struct {
	apiKey   string
	baseURL  string
	client   *http.Client
}

func NewPagerDutyProvider(apiKey, baseURL string) *PagerDutyProvider {
	return &PagerDutyProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *PagerDutyProvider) Name() string { return "pagerduty" }

// Deliver sends a PagerDuty trigger event for each recipient with a routing key.
func (p *PagerDutyProvider) Deliver(
	ctx context.Context,
	req *models.NotificationRequest,
	recipient models.Recipient,
) (string, error) {
	ctx, span := pdTracer.Start(ctx, "pagerduty.deliver")
	defer span.End()
	span.SetAttributes(
		attribute.String("recipient.id", recipient.RecipientID),
		attribute.String("crisis.id", req.CrisisID),
	)

	routingKey := recipient.PagerDutyKey
	if routingKey == "" {
		routingKey = p.apiKey // fall back to global routing key
	}

	severity := mapSeverity(req.Severity)
	pd := PagerDutyPayload{
		RoutingKey:  routingKey,
		EventAction: "trigger",
		DedupKey:    req.IdempotencyKey,
		Client:      "MEDUSA/SMA",
		Payload: PDInnerPayload{
			Summary:   fmt.Sprintf("[%s] %s — %s", req.Severity, req.Classification, req.Message),
			Source:    fmt.Sprintf("hospital/%s", req.HospitalID),
			Severity:  severity,
			Timestamp: req.CreatedAt.Format(time.RFC3339),
			CustomDetails: map[string]string{
				"crisis_id":   req.CrisisID,
				"hospital_id": req.HospitalID,
				"trace_id":    req.TraceID,
			},
		},
	}

	body, err := json.Marshal(pd)
	if err != nil {
		return "", fmt.Errorf("pagerduty marshal failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Routing-Key", routingKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("pagerduty request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pagerduty returned %d", resp.StatusCode)
	}

	return fmt.Sprintf("pd-dedup:%s", req.IdempotencyKey), nil
}

func mapSeverity(s string) string {
	switch s {
	case "CRITICAL":
		return "critical"
	case "HIGH":
		return "error"
	case "MEDIUM":
		return "warning"
	default:
		return "info"
	}
}

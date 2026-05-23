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

var pushTracer = otel.Tracer("messaging-agent/push")

// PushProvider delivers APNs (iOS) and FCM (Android) push notifications.
// In production, APNs uses JWT auth via the key file; FCM uses the server key.
type PushProvider struct {
	apnsKeyPath  string
	apnsTeamID   string
	apnsBundleID string
	fcmServerKey string
	fcmBaseURL   string
	client       *http.Client
}

func NewPushProvider(apnsKeyPath, apnsTeamID, apnsBundleID, fcmServerKey string) *PushProvider {
	return &PushProvider{
		apnsKeyPath:  apnsKeyPath,
		apnsTeamID:   apnsTeamID,
		apnsBundleID: apnsBundleID,
		fcmServerKey: fcmServerKey,
		fcmBaseURL:   "https://fcm.googleapis.com/fcm/send",
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *PushProvider) Name() string { return "push" }

// Deliver routes to APNs or FCM based on recipient platform.
func (p *PushProvider) Deliver(
	ctx context.Context,
	req *models.NotificationRequest,
	recipient models.Recipient,
) (string, error) {
	ctx, span := pushTracer.Start(ctx, "push.deliver")
	defer span.End()
	span.SetAttributes(
		attribute.String("platform", recipient.Platform),
		attribute.String("recipient.id", recipient.RecipientID),
	)

	if recipient.DeviceToken == "" {
		return "", fmt.Errorf("recipient %s has no device token", recipient.RecipientID)
	}

	switch recipient.Platform {
	case "ios":
		return p.deliverAPNs(ctx, req, recipient)
	case "android":
		return p.deliverFCM(ctx, req, recipient)
	default:
		return "", fmt.Errorf("unknown platform: %s", recipient.Platform)
	}
}

// deliverAPNs sends an APNs push notification via HTTP/2 provider API.
// Full implementation requires apple JWT signing; this stub uses the HTTP wrapper.
func (p *PushProvider) deliverAPNs(ctx context.Context, req *models.NotificationRequest, recipient models.Recipient) (string, error) {
	endpoint := fmt.Sprintf("https://api.push.apple.com/3/device/%s", recipient.DeviceToken)
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]string{
				"title": fmt.Sprintf("[MEDUSA] %s Alert", req.Severity),
				"body":  req.Message,
			},
			"sound": "default",
			"badge": 1,
		},
		"crisis_id":   req.CrisisID,
		"hospital_id": req.HospitalID,
		"trace_id":    req.TraceID,
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apns-topic", p.apnsBundleID)
	httpReq.Header.Set("apns-priority", "10")
	// TODO: add JWT auth header: httpReq.Header.Set("Authorization", "bearer "+signedJWT)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("APNs request failed: %w", err)
	}
	defer resp.Body.Close()
	apnsID := resp.Header.Get("apns-id")
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("APNs returned %d", resp.StatusCode)
	}
	return apnsID, nil
}

// deliverFCM sends a Firebase Cloud Messaging notification.
func (p *PushProvider) deliverFCM(ctx context.Context, req *models.NotificationRequest, recipient models.Recipient) (string, error) {
	payload := map[string]interface{}{
		"to": recipient.DeviceToken,
		"notification": map[string]string{
			"title": fmt.Sprintf("[MEDUSA] %s Alert", req.Severity),
			"body":  req.Message,
		},
		"data": map[string]string{
			"crisis_id":   req.CrisisID,
			"hospital_id": req.HospitalID,
			"trace_id":    req.TraceID,
		},
		"priority": "high",
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.fcmBaseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "key="+p.fcmServerKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("FCM request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FCM returned %d", resp.StatusCode)
	}
	var result struct {
		MessageID string `json:"message_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.MessageID, nil
}

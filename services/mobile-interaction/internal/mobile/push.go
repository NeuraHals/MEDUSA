package mobile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/antigravity/mono/services/mobile-interaction/internal/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var pushTracer = otel.Tracer("mobile-interaction/push")

// PushClient delivers approval prompt notifications via APNs and FCM.
type PushClient struct {
	log          *zap.Logger
	apnsKeyPath  string
	apnsTeamID   string
	apnsBundleID string
	fcmServerKey string
	fcmBaseURL   string
	client       *http.Client
}

func NewPushClient(
	log *zap.Logger,
	apnsKeyPath, apnsTeamID, apnsBundleID, fcmServerKey string,
) *PushClient {
	return &PushClient{
		log:          log,
		apnsKeyPath:  apnsKeyPath,
		apnsTeamID:   apnsTeamID,
		apnsBundleID: apnsBundleID,
		fcmServerKey: fcmServerKey,
		fcmBaseURL:   "https://fcm.googleapis.com/fcm/send",
		client:       &http.Client{Timeout: 8 * time.Second},
	}
}

// SendApprovalPrompt dispatches a critical push notification to the operator's device.
func (p *PushClient) SendApprovalPrompt(ctx context.Context, req *models.PushNotificationRequest) error {
	ctx, span := pushTracer.Start(ctx, "push.sendApprovalPrompt")
	defer span.End()
	span.SetAttributes(
		attribute.String("operator.id", req.OperatorID),
		attribute.String("platform", req.Platform),
		attribute.String("blueprint.id", req.BlueprintID),
	)

	switch req.Platform {
	case "ios":
		return p.sendAPNs(ctx, req)
	case "android":
		return p.sendFCM(ctx, req)
	default:
		return fmt.Errorf("unsupported platform: %s", req.Platform)
	}
}

func (p *PushClient) sendAPNs(ctx context.Context, req *models.PushNotificationRequest) error {
	endpoint := fmt.Sprintf("https://api.push.apple.com/3/device/%s", req.DeviceToken)
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]string{
				"title": fmt.Sprintf("[MEDUSA] Approval Required — PRI %d", req.PRIScore),
				"body":  req.Message,
			},
			"sound":    "critical.aiff",
			"badge":    1,
			"category": "MEDUSA_APPROVAL",
		},
		"blueprint_id":   req.BlueprintID,
		"action_id":      req.ActionID,
		"hospital_id":    req.HospitalID,
		"trace_id":       req.TraceID,
		"idempotency_key": req.IdempotencyKey,
		"expires_at":     req.ExpiresAt.Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apns-topic", p.apnsBundleID)
	httpReq.Header.Set("apns-priority", "10")     // critical priority
	httpReq.Header.Set("apns-push-type", "alert")
	// TODO: add JWT bearer: "Authorization", "bearer "+signedJWT
	// Production: sign JWT using APNs p8 key at APNsKeyPath

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("APNs request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("APNs returned %d for device %s", resp.StatusCode, req.DeviceToken)
	}
	p.log.Info("APNs push delivered",
		zap.String("blueprint_id", req.BlueprintID),
		zap.String("apns_id", resp.Header.Get("apns-id")),
	)
	return nil
}

func (p *PushClient) sendFCM(ctx context.Context, req *models.PushNotificationRequest) error {
	payload := map[string]interface{}{
		"to":       req.DeviceToken,
		"priority": "high",
		"notification": map[string]string{
			"title": fmt.Sprintf("[MEDUSA] Approval Required — PRI %d", req.PRIScore),
			"body":  req.Message,
		},
		"data": map[string]string{
			"blueprint_id":    req.BlueprintID,
			"action_id":       req.ActionID,
			"hospital_id":     req.HospitalID,
			"trace_id":        req.TraceID,
			"idempotency_key": req.IdempotencyKey,
			"expires_at":      req.ExpiresAt.Format(time.RFC3339),
		},
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.fcmBaseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "key="+p.fcmServerKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("FCM request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FCM returned %d", resp.StatusCode)
	}
	p.log.Info("FCM push delivered",
		zap.String("blueprint_id", req.BlueprintID),
		zap.String("operator_id", req.OperatorID),
	)
	return nil
}

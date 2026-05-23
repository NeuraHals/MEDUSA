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

var twilioTracer = otel.Tracer("messaging-agent/twilio")

// TwilioProvider delivers SMS messages via the Twilio REST API.
type TwilioProvider struct {
	accountSID string
	authToken  string
	fromNumber string
	baseURL    string
	client     *http.Client
}

func NewTwilioProvider(accountSID, authToken, fromNumber, baseURL string) *TwilioProvider {
	return &TwilioProvider{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TwilioProvider) Name() string { return "twilio" }

// Deliver sends an SMS to a recipient's registered phone number.
func (t *TwilioProvider) Deliver(
	ctx context.Context,
	req *models.NotificationRequest,
	recipient models.Recipient,
) (string, error) {
	ctx, span := twilioTracer.Start(ctx, "twilio.deliver")
	defer span.End()
	span.SetAttributes(
		attribute.String("recipient.id", recipient.RecipientID),
		attribute.String("recipient.role", recipient.Role),
	)

	if recipient.PhoneNumber == "" {
		return "", fmt.Errorf("recipient %s has no phone number", recipient.RecipientID)
	}

	body := fmt.Sprintf(
		"[MEDUSA ALERT] %s\nHospital: %s\nSeverity: %s\nRef: %s",
		req.Message, req.HospitalID, req.Severity, req.CrisisID,
	)

	endpoint := fmt.Sprintf("%s/Accounts/%s/Messages.json", t.baseURL, t.accountSID)
	payload := map[string]string{
		"To":   recipient.PhoneNumber,
		"From": t.fromNumber,
		"Body": body,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	httpReq.SetBasicAuth(t.accountSID, t.authToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("twilio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("twilio returned %d", resp.StatusCode)
	}

	var result struct {
		SID string `json:"sid"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result.SID, nil
}

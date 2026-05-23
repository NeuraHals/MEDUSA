package mobile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/antigravity/mono/services/mobile-interaction/internal/models"
	"go.uber.org/zap"
)

// SMSClient provides SMS-based degraded-mode fallback for offline operators.
type SMSClient struct {
	log        *zap.Logger
	accountSID string
	authToken  string
	fromNumber string
	baseURL    string
	client     *http.Client
}

func NewSMSClient(log *zap.Logger, accountSID, authToken, fromNumber, baseURL string) *SMSClient {
	return &SMSClient{
		log:        log,
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 8 * time.Second},
	}
}

// SendApprovalSMS sends an SMS approval prompt as a degraded-mode fallback
// when push notification delivery fails or the operator is offline.
func (s *SMSClient) SendApprovalSMS(ctx context.Context, session *models.MobileSession, entry *models.OfflineApprovalEntry) error {
	if session.PhoneNumber == "" {
		return fmt.Errorf("operator %s has no phone number for SMS fallback", session.OperatorID)
	}

	body := fmt.Sprintf(
		"[MEDUSA URGENT] Approval required.\nBlueprint: %s\nHospital: %s\nPRI: %d\nExpires: %s\nReply via MEDUSA app or call SOC.",
		entry.BlueprintID, entry.HospitalID, entry.PRIScore,
		entry.ExpiresAt.Format(time.RFC3339),
	)

	endpoint := fmt.Sprintf("%s/Accounts/%s/Messages.json", s.baseURL, s.accountSID)
	payload := map[string]string{
		"To":   session.PhoneNumber,
		"From": s.fromNumber,
		"Body": body,
	}

	encoded, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.accountSID, s.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("SMS request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned %d", resp.StatusCode)
	}

	s.log.Info("SMS fallback sent",
		zap.String("operator_id", session.OperatorID),
		zap.String("blueprint_id", entry.BlueprintID),
	)
	return nil
}

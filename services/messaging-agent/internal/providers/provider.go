package providers

import (
	"context"

	"github.com/antigravity/mono/services/messaging-agent/internal/models"
)

// Provider is the common interface for all messaging backends.
type Provider interface {
	Name() string
	Deliver(ctx context.Context, req *models.NotificationRequest, recipient models.Recipient) (string, error)
}

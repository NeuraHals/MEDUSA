package mobile

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/mobile-interaction/internal/models"
	"github.com/antigravity/mono/services/mobile-interaction/internal/state"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var reconcileTracer = otel.Tracer("mobile-interaction/reconciler")

// Reconciler handles device state sync, offline queue draining,
// and degraded-mode SMS escalation.
type Reconciler struct {
	log          *zap.Logger
	redis        *state.RedisClient
	pushClient   *PushClient
	smsClient    *SMSClient
	offlineTTL   int
	sessionTTL   int
	degradedMode bool
}

func NewReconciler(
	log *zap.Logger,
	redis *state.RedisClient,
	pushClient *PushClient,
	smsClient *SMSClient,
	offlineTTL, sessionTTL int,
	degradedMode bool,
) *Reconciler {
	return &Reconciler{
		log:          log,
		redis:        redis,
		pushClient:   pushClient,
		smsClient:    smsClient,
		offlineTTL:   offlineTTL,
		sessionTTL:   sessionTTL,
		degradedMode: degradedMode,
	}
}

// RegisterDevice stores or refreshes a mobile operator session.
func (r *Reconciler) RegisterDevice(ctx context.Context, operatorID, deviceToken, platform, phoneNumber string) error {
	session := &models.MobileSession{
		OperatorID:  operatorID,
		DeviceToken: deviceToken,
		Platform:    platform,
		Online:      true,
		LastSeenAt:  time.Now().UTC(),
		PhoneNumber: phoneNumber,
	}
	if err := r.redis.SetSession(ctx, session, r.sessionTTL); err != nil {
		return fmt.Errorf("register device failed: %w", err)
	}
	r.log.Info("device registered",
		zap.String("operator_id", operatorID),
		zap.String("platform", platform),
	)
	// Drain any pending offline approvals that arrived while device was offline
	return r.drainOfflineQueue(ctx, operatorID)
}

// SendApprovalPrompt delivers a push notification or queues offline if device is unreachable.
func (r *Reconciler) SendApprovalPrompt(ctx context.Context, req *models.PushNotificationRequest) error {
	ctx, span := reconcileTracer.Start(ctx, "reconciler.sendApprovalPrompt")
	defer span.End()
	span.SetAttributes(
		attribute.String("operator.id", req.OperatorID),
		attribute.String("blueprint.id", req.BlueprintID),
	)

	session, err := r.redis.GetSession(ctx, req.OperatorID)
	if err != nil {
		return err
	}

	// In degraded mode, skip push and go direct to SMS
	if r.degradedMode {
		return r.escalateToSMS(ctx, req, session)
	}

	if session == nil || !session.Online || session.DeviceToken == "" {
		r.log.Warn("operator offline — queueing approval",
			zap.String("operator_id", req.OperatorID),
			zap.String("blueprint_id", req.BlueprintID),
		)
		return r.enqueueOffline(ctx, req)
	}

	// Populate device token from session if not set in request
	if req.DeviceToken == "" {
		req.DeviceToken = session.DeviceToken
		req.Platform = session.Platform
	}

	if err := r.pushClient.SendApprovalPrompt(ctx, req); err != nil {
		r.log.Error("push failed — queueing offline",
			zap.Error(err),
			zap.String("operator_id", req.OperatorID),
		)
		// Mark device offline and queue for SMS fallback
		_ = r.redis.MarkOperatorOnline(ctx, req.OperatorID, false)
		return r.escalateToSMS(ctx, req, session)
	}

	return nil
}

func (r *Reconciler) enqueueOffline(ctx context.Context, req *models.PushNotificationRequest) error {
	entry := &models.OfflineApprovalEntry{
		EntryID:        uuid.New().String(),
		BlueprintID:    req.BlueprintID,
		ActionID:       req.ActionID,
		HospitalID:     req.HospitalID,
		PRIScore:       req.PRIScore,
		Classification: req.Classification,
		Message:        req.Message,
		OperatorID:     req.OperatorID,
		TraceID:        req.TraceID,
		IdempotencyKey: req.IdempotencyKey,
		QueuedAt:       time.Now().UTC(),
		ExpiresAt:      req.ExpiresAt,
	}
	return r.redis.EnqueueOfflineApproval(ctx, entry, r.offlineTTL)
}

func (r *Reconciler) drainOfflineQueue(ctx context.Context, operatorID string) error {
	entries, err := r.redis.DequeueOfflineApprovals(ctx, operatorID)
	if err != nil || len(entries) == 0 {
		return err
	}

	session, _ := r.redis.GetSession(ctx, operatorID)
	if session == nil {
		return nil
	}

	for _, entry := range entries {
		// Skip expired entries
		if time.Now().After(entry.ExpiresAt) {
			r.log.Warn("offline entry expired — discarding",
				zap.String("entry_id", entry.EntryID),
				zap.String("blueprint_id", entry.BlueprintID),
			)
			continue
		}
		pushReq := &models.PushNotificationRequest{
			RequestID:      uuid.New().String(),
			BlueprintID:    entry.BlueprintID,
			ActionID:       entry.ActionID,
			HospitalID:     entry.HospitalID,
			PRIScore:       entry.PRIScore,
			Classification: entry.Classification,
			Message:        entry.Message,
			OperatorID:     entry.OperatorID,
			DeviceToken:    session.DeviceToken,
			Platform:       session.Platform,
			ExpiresAt:      entry.ExpiresAt,
			TraceID:        entry.TraceID,
			IdempotencyKey: entry.IdempotencyKey,
		}
		if err := r.pushClient.SendApprovalPrompt(ctx, pushReq); err != nil {
			r.log.Error("drain push failed", zap.Error(err), zap.String("blueprint_id", entry.BlueprintID))
		}
	}
	return nil
}

func (r *Reconciler) escalateToSMS(ctx context.Context, req *models.PushNotificationRequest, session *models.MobileSession) error {
	if session == nil || session.PhoneNumber == "" {
		r.log.Warn("no phone number for SMS fallback — queueing offline only",
			zap.String("operator_id", req.OperatorID),
		)
		return r.enqueueOffline(ctx, req)
	}

	entry := &models.OfflineApprovalEntry{
		BlueprintID:    req.BlueprintID,
		HospitalID:     req.HospitalID,
		PRIScore:       req.PRIScore,
		OperatorID:     req.OperatorID,
		ExpiresAt:      req.ExpiresAt,
	}
	return r.smsClient.SendApprovalSMS(ctx, session, entry)
}

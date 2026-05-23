package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/antigravity/mono/services/audit-agent/internal/models"
	"go.uber.org/zap"
)

// WORMClient persists audit events to S3 with Object Lock enabled.
// Object Lock in COMPLIANCE mode prevents any entity — including root — from
// deleting or modifying objects before the retention period expires.
type WORMClient struct {
	log            *zap.Logger
	client         *s3.Client
	bucket         string
	prefix         string
	objectLockMode string
}

func NewWORMClient(ctx context.Context, log *zap.Logger, region, bucket, prefix, objectLockMode string) (*WORMClient, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS config load failed: %w", err)
	}
	return &WORMClient{
		log:            log,
		client:         s3.NewFromConfig(cfg),
		bucket:         bucket,
		prefix:         prefix,
		objectLockMode: objectLockMode,
	}, nil
}

// PersistEvent writes a single audit event as an immutable S3 object with Object Lock.
// The object key encodes the hospital, crisis, date, and event ID for forensic retrieval.
// Returns the S3 key on success.
func (w *WORMClient) PersistEvent(ctx context.Context, event *models.AuditEvent, retainUntil time.Time) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("serialise failed: %w", err)
	}

	key := w.buildKey(event)

	// Determine Object Lock mode
	var lockMode types.ObjectLockMode
	if w.objectLockMode == "GOVERNANCE" {
		lockMode = types.ObjectLockModeGovernance
	} else {
		lockMode = types.ObjectLockModeCompliance
	}

	retainUntilPtr := retainUntil
	input := &s3.PutObjectInput{
		Bucket:                    aws.String(w.bucket),
		Key:                       aws.String(key),
		Body:                      bytes.NewReader(data),
		ContentType:               aws.String("application/json"),
		ObjectLockMode:            lockMode,
		ObjectLockRetainUntilDate: &retainUntilPtr,
		// Immutability metadata
		Metadata: map[string]string{
			"event-id":        event.EventID,
			"event-type":      string(event.EventType),
			"crisis-id":       event.CrisisID,
			"hospital-id":     event.HospitalID,
			"event-hash":      event.EventHash,
			"retention-class": string(event.RetentionClass),
			"legal-hold":      fmt.Sprintf("%v", event.LegalHold),
		},
	}

	// If legal hold is set, apply S3 Object Lock legal hold tag as well
	if event.LegalHold {
		lhInput := &s3.PutObjectLegalHoldInput{
			Bucket: aws.String(w.bucket),
			Key:    aws.String(key),
			LegalHold: &types.ObjectLockLegalHold{
				Status: types.ObjectLockLegalHoldStatusOn,
			},
		}
		defer func() {
			_, _ = w.client.PutObjectLegalHold(ctx, lhInput)
		}()
	}

	_, err = w.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("S3 PutObject failed for key %s: %w", key, err)
	}

	w.log.Info("audit event persisted to WORM storage",
		zap.String("s3_key", key),
		zap.String("event_id", event.EventID),
		zap.String("retention_class", string(event.RetentionClass)),
		zap.Time("retain_until", retainUntil),
	)

	return key, nil
}

// buildKey generates a deterministic, hierarchical S3 key for the event.
// Format: {prefix}{hospital_id}/{crisis_id}/{year}/{month}/{day}/{event_type}/{event_id}.json
func (w *WORMClient) buildKey(event *models.AuditEvent) string {
	t := event.OccurredAt.UTC()
	return fmt.Sprintf("%s%s/%s/%04d/%02d/%02d/%s/%s.json",
		w.prefix,
		event.HospitalID,
		event.CrisisID,
		t.Year(), t.Month(), t.Day(),
		string(event.EventType),
		event.EventID,
	)
}

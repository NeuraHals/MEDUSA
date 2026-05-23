package replay

import (
	"context"
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/models"
	"github.com/antigravity/mono/services/audit-agent/internal/state"
	"go.uber.org/zap"
)

// Indexer builds and queries the hot replay index stored in Redis.
// The hot index covers the most recent 90 days; older events require S3 direct queries.
type Indexer struct {
	log   *zap.Logger
	redis *state.RedisClient
}

func NewIndexer(log *zap.Logger, redis *state.RedisClient) *Indexer {
	return &Indexer{log: log, redis: redis}
}

// Index adds a new audit event to the replay index.
func (i *Indexer) Index(ctx context.Context, event *models.AuditEvent, s3Key string) error {
	entry := &models.ReplayIndexEntry{
		EventID:     event.EventID,
		EventType:   event.EventType,
		CrisisID:    event.CrisisID,
		BlueprintID: event.BlueprintID,
		HospitalID:  event.HospitalID,
		S3Key:       s3Key,
		EventHash:   event.EventHash,
		OccurredAt:  event.OccurredAt,
	}
	if err := i.redis.IndexEvent(ctx, entry); err != nil {
		i.log.Error("replay index write failed",
			zap.Error(err),
			zap.String("event_id", event.EventID),
		)
		return err
	}
	return nil
}

// Query returns all events for a crisis in the given time window, ordered by occurrence.
func (i *Indexer) Query(ctx context.Context, crisisID string, from, to time.Time) ([]models.ReplayIndexEntry, error) {
	return i.redis.QueryReplayIndex(ctx, crisisID, from, to)
}

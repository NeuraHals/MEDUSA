package approval

import (
	"context"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/antigravity/mono/services/oversight-agent/internal/state"
	"go.uber.org/zap"
)

const glassBreakTimeoutSecs = 60

// TimeoutManager monitors pending approvals and triggers Glass Break escalation.
type TimeoutManager struct {
	log      *zap.Logger
	redis    *state.RedisClient
	timeoutCh chan string // receives blueprint_id on timeout
	stopCh   chan struct{}
}

func NewTimeoutManager(log *zap.Logger, redis *state.RedisClient) *TimeoutManager {
	return &TimeoutManager{
		log:      log,
		redis:    redis,
		timeoutCh: make(chan string, 64),
		stopCh:   make(chan struct{}),
	}
}

// Schedule registers a new approval for timeout monitoring.
// After glassBreakTimeoutSecs, it triggers automatic escalation.
func (t *TimeoutManager) Schedule(ctx context.Context, blueprintID string) {
	go func() {
		timer := time.NewTimer(glassBreakTimeoutSecs * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
			t.log.Warn("approval timeout — triggering Glass Break escalation",
				zap.String("blueprint_id", blueprintID),
				zap.Int("timeout_secs", glassBreakTimeoutSecs),
			)
			// Check if it's still pending (could have been approved/denied already)
			record, err := t.redis.GetApprovalRecord(ctx, blueprintID)
			if err != nil || record == nil {
				return
			}
			if record.Status != models.StatusPending {
				return // already resolved — no Glass Break needed
			}
			select {
			case t.timeoutCh <- blueprintID:
			default:
				t.log.Error("timeout channel full — Glass Break dropped",
					zap.String("blueprint_id", blueprintID),
				)
			}
		case <-t.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}()
}

// TimeoutCh exposes the timeout signal channel for the approval service to consume.
func (t *TimeoutManager) TimeoutCh() <-chan string {
	return t.timeoutCh
}

func (t *TimeoutManager) Stop() {
	close(t.stopCh)
}

package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/mobile-interaction/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyTTL = 24 * time.Hour
)

// RedisClient wraps go-redis for MIA-specific state operations.
type RedisClient struct {
	rdb *redis.Client
}

func NewRedisClient(redisURL string) (*RedisClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RedisClient{rdb: rdb}, nil
}

// --- Session state ---

// SetSession stores or updates a mobile operator session.
func (r *RedisClient) SetSession(ctx context.Context, session *models.MobileSession, ttlSecs int) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	key := sessionKey(session.OperatorID)
	return r.rdb.Set(ctx, key, data, time.Duration(ttlSecs)*time.Second).Err()
}

// GetSession retrieves an operator session. Returns nil if not found.
func (r *RedisClient) GetSession(ctx context.Context, operatorID string) (*models.MobileSession, error) {
	data, err := r.rdb.Get(ctx, sessionKey(operatorID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var session models.MobileSession
	return &session, json.Unmarshal(data, &session)
}

// MarkOperatorOnline updates the online flag in an existing session.
func (r *RedisClient) MarkOperatorOnline(ctx context.Context, operatorID string, online bool) error {
	session, err := r.GetSession(ctx, operatorID)
	if err != nil || session == nil {
		return err
	}
	session.Online = online
	session.LastSeenAt = time.Now().UTC()
	return r.SetSession(ctx, session, 3600)
}

// --- Offline queue ---

// EnqueueOfflineApproval pushes a pending approval to the operator's offline queue.
func (r *RedisClient) EnqueueOfflineApproval(ctx context.Context, entry *models.OfflineApprovalEntry, ttlSecs int) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	key := offlineQueueKey(entry.OperatorID)
	pipe := r.rdb.Pipeline()
	pipe.LPush(ctx, key, data)
	pipe.Expire(ctx, key, time.Duration(ttlSecs)*time.Second)
	_, err = pipe.Exec(ctx)
	return err
}

// DequeueOfflineApprovals returns and clears all pending offline entries for an operator.
func (r *RedisClient) DequeueOfflineApprovals(ctx context.Context, operatorID string) ([]*models.OfflineApprovalEntry, error) {
	key := offlineQueueKey(operatorID)
	items, err := r.rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	_ = r.rdb.Del(ctx, key)
	entries := make([]*models.OfflineApprovalEntry, 0, len(items))
	for _, item := range items {
		var entry models.OfflineApprovalEntry
		if err := json.Unmarshal([]byte(item), &entry); err == nil {
			entries = append(entries, &entry)
		}
	}
	return entries, nil
}

// --- Idempotency ---

// IsApprovalDuplicate returns true if this idempotency key was already processed.
func (r *RedisClient) IsApprovalDuplicate(ctx context.Context, key string) bool {
	val, err := r.rdb.Exists(ctx, idempotencyKey(key)).Result()
	if err != nil {
		return false
	}
	return val > 0
}

// MarkApprovalProcessed records an idempotency key with 24h TTL.
func (r *RedisClient) MarkApprovalProcessed(ctx context.Context, key string) error {
	return r.rdb.Set(ctx, idempotencyKey(key), "processed", idempotencyTTL).Err()
}

func sessionKey(operatorID string) string   { return fmt.Sprintf("mia:session:%s", operatorID) }
func offlineQueueKey(operatorID string) string { return fmt.Sprintf("mia:offline:%s", operatorID) }
func idempotencyKey(key string) string      { return fmt.Sprintf("mia:idem:%s", key) }

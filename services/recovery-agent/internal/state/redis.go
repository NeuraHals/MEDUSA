package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/recovery-agent/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	rollbackTTL    = 48 * time.Hour
	idempotencyTTL = 24 * time.Hour
)

// RedisClient manages rollback record persistence and idempotency.
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

// StoreRecord persists a rollback record with a 48-hour TTL.
func (r *RedisClient) StoreRecord(ctx context.Context, record *models.RollbackRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, rollbackKey(record.RollbackID), data, rollbackTTL).Err()
}

// GetRecord retrieves a rollback record. Returns nil if not found.
func (r *RedisClient) GetRecord(ctx context.Context, rollbackID string) (*models.RollbackRecord, error) {
	data, err := r.rdb.Get(ctx, rollbackKey(rollbackID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var record models.RollbackRecord
	return &record, json.Unmarshal(data, &record)
}

// SetState atomically updates the rollback state field.
func (r *RedisClient) SetState(ctx context.Context, rollbackID string, state models.RollbackState) error {
	record, err := r.GetRecord(ctx, rollbackID)
	if err != nil || record == nil {
		return fmt.Errorf("record not found: %s", rollbackID)
	}
	record.State = state
	if state == models.RollbackCompleted || state == models.RollbackFailed || state == models.RollbackPartial {
		now := time.Now().UTC()
		record.CompletedAt = &now
	}
	return r.StoreRecord(ctx, record)
}

// IsAlreadyRolledBack checks idempotency — returns true if this rollback has been processed.
func (r *RedisClient) IsAlreadyRolledBack(ctx context.Context, rollbackID string) bool {
	val, err := r.rdb.Exists(ctx, idemKey(rollbackID)).Result()
	if err != nil { return false }
	return val > 0
}

// MarkRolledBack records a completed rollback to prevent replay.
func (r *RedisClient) MarkRolledBack(ctx context.Context, rollbackID string) error {
	return r.rdb.Set(ctx, idemKey(rollbackID), "done", idempotencyTTL).Err()
}

func rollbackKey(id string) string { return fmt.Sprintf("recovery:rollback:%s", id) }
func idemKey(id string) string     { return fmt.Sprintf("recovery:idem:%s", id) }

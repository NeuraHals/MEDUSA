package state

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	idempotencyTTL = 24 * time.Hour
	executionTTL   = 24 * time.Hour
)

// RedisClient wraps go-redis with idempotency and execution state helpers.
type RedisClient struct {
	rdb *redis.Client
}

// NewRedisClient connects to Redis and verifies connectivity.
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

// IsAlreadyExecuted returns true if a blueprint has already been executed.
// Fail-safe: returns false on Redis error to allow processing to continue.
func (r *RedisClient) IsAlreadyExecuted(ctx context.Context, blueprintID string) bool {
	key := fmt.Sprintf("executed:blueprint:%s", blueprintID)
	val, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false // fail-safe: do not block on Redis failure
	}
	return val > 0
}

// MarkExecuted records a blueprint as executed with 24h TTL.
func (r *RedisClient) MarkExecuted(ctx context.Context, blueprintID string) error {
	key := fmt.Sprintf("executed:blueprint:%s", blueprintID)
	return r.rdb.Set(ctx, key, "executed", idempotencyTTL).Err()
}

// SetExecutionState persists the current AOA state machine state.
func (r *RedisClient) SetExecutionState(ctx context.Context, blueprintID, state string) error {
	key := fmt.Sprintf("state:execution:%s", blueprintID)
	return r.rdb.Set(ctx, key, state, executionTTL).Err()
}

// GetExecutionState retrieves the current state for a blueprint.
func (r *RedisClient) GetExecutionState(ctx context.Context, blueprintID string) (string, error) {
	key := fmt.Sprintf("state:execution:%s", blueprintID)
	return r.rdb.Get(ctx, key).Result()
}

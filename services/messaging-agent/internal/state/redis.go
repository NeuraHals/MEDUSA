package state

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const deliveryTTL = 24 * time.Hour

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

// IsAlreadyDelivered returns true if the delivery idempotency key has been seen.
// Fail-safe: returns false on Redis error (allows delivery attempt).
func (r *RedisClient) IsAlreadyDelivered(ctx context.Context, key string) bool {
	val, err := r.rdb.Exists(ctx, deliveryKey(key)).Result()
	if err != nil {
		return false
	}
	return val > 0
}

// MarkDelivered records a successful delivery with a 24h TTL.
func (r *RedisClient) MarkDelivered(ctx context.Context, key string) error {
	return r.rdb.Set(ctx, deliveryKey(key), "delivered", deliveryTTL).Err()
}

func deliveryKey(key string) string {
	return fmt.Sprintf("delivery:idem:%s", key)
}

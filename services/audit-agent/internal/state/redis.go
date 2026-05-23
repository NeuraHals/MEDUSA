package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/audit-agent/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyTTL = 24 * time.Hour
	chainHeadTTL   = 30 * 24 * time.Hour // 30-day active chain head
	replayIndexTTL = 90 * 24 * time.Hour // 90-day hot replay index
)

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

// --- Idempotency ---

func (r *RedisClient) IsAlreadyAudited(ctx context.Context, eventID string) bool {
	val, err := r.rdb.Exists(ctx, auditIdemKey(eventID)).Result()
	if err != nil { return false }
	return val > 0
}

func (r *RedisClient) MarkAudited(ctx context.Context, eventID string) error {
	return r.rdb.Set(ctx, auditIdemKey(eventID), "audited", idempotencyTTL).Err()
}

// --- Chain head management ---
// The chain head is the hash of the most-recently-persisted event for a crisis.
// Each new event links back to this hash, forming a tamper-evident chain.

func (r *RedisClient) GetChainHead(ctx context.Context, crisisID string) (string, error) {
	val, err := r.rdb.Get(ctx, chainHeadKey(crisisID)).Result()
	if err == redis.Nil {
		return "", nil // genesis — no previous event
	}
	return val, err
}

func (r *RedisClient) SetChainHead(ctx context.Context, crisisID, hash string) error {
	return r.rdb.Set(ctx, chainHeadKey(crisisID), hash, chainHeadTTL).Err()
}

// GetChainLength returns the current length of the audit chain for a crisis.
func (r *RedisClient) IncrChainLength(ctx context.Context, crisisID string) (int64, error) {
	return r.rdb.Incr(ctx, chainLengthKey(crisisID)).Result()
}

// --- Replay index ---

func (r *RedisClient) IndexEvent(ctx context.Context, entry *models.ReplayIndexEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	// Sorted set keyed by epoch for time-range queries
	score := float64(entry.OccurredAt.UnixNano())
	pipe := r.rdb.Pipeline()
	// Per-crisis index
	pipe.ZAdd(ctx, replayIndexKey(entry.CrisisID), redis.Z{Score: score, Member: string(data)})
	pipe.Expire(ctx, replayIndexKey(entry.CrisisID), replayIndexTTL)
	// Per-hospital index
	pipe.ZAdd(ctx, hospitalIndexKey(entry.HospitalID), redis.Z{Score: score, Member: entry.EventID})
	pipe.Expire(ctx, hospitalIndexKey(entry.HospitalID), replayIndexTTL)
	_, err = pipe.Exec(ctx)
	return err
}

// QueryReplayIndex retrieves replay entries for a crisis in time order.
func (r *RedisClient) QueryReplayIndex(ctx context.Context, crisisID string, from, to time.Time) ([]models.ReplayIndexEntry, error) {
	members, err := r.rdb.ZRangeByScore(ctx, replayIndexKey(crisisID), &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", from.UnixNano()),
		Max: fmt.Sprintf("%d", to.UnixNano()),
	}).Result()
	if err != nil {
		return nil, err
	}
	entries := make([]models.ReplayIndexEntry, 0, len(members))
	for _, m := range members {
		var entry models.ReplayIndexEntry
		if err := json.Unmarshal([]byte(m), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func auditIdemKey(eventID string) string   { return fmt.Sprintf("audit:idem:%s", eventID) }
func chainHeadKey(crisisID string) string  { return fmt.Sprintf("audit:chain:head:%s", crisisID) }
func chainLengthKey(crisisID string) string { return fmt.Sprintf("audit:chain:len:%s", crisisID) }
func replayIndexKey(crisisID string) string { return fmt.Sprintf("audit:replay:%s", crisisID) }
func hospitalIndexKey(hID string) string   { return fmt.Sprintf("audit:hospital:%s", hID) }

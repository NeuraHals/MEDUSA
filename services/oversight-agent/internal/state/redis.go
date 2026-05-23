package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antigravity/mono/services/oversight-agent/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	approvalTTL   = 24 * time.Hour
	idempotencyTTL = 24 * time.Hour
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

// StoreApprovalRecord persists a pending approval record with TTL.
func (r *RedisClient) StoreApprovalRecord(ctx context.Context, record *models.ApprovalRecord) error {
	key := approvalKey(record.Request.BlueprintID)
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}
	return r.rdb.Set(ctx, key, data, approvalTTL).Err()
}

// GetApprovalRecord retrieves a pending approval record. Returns nil if not found.
func (r *RedisClient) GetApprovalRecord(ctx context.Context, blueprintID string) (*models.ApprovalRecord, error) {
	key := approvalKey(blueprintID)
	data, err := r.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var record models.ApprovalRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateApprovalStatus atomically updates the status of a stored approval record.
func (r *RedisClient) UpdateApprovalStatus(ctx context.Context, blueprintID string, status models.ApprovalStatus, approverID string) error {
	record, err := r.GetApprovalRecord(ctx, blueprintID)
	if err != nil || record == nil {
		return fmt.Errorf("approval record not found: %s", blueprintID)
	}
	record.Status = status
	record.ApproverID = approverID
	now := time.Now().UTC()
	record.ProcessedAt = &now
	return r.StoreApprovalRecord(ctx, record)
}

// IsApprovalDuplicate returns true if this idempotency key has already been processed.
func (r *RedisClient) IsApprovalDuplicate(ctx context.Context, idempotencyKey string) bool {
	key := fmt.Sprintf("approval:idem:%s", idempotencyKey)
	val, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false // fail-safe
	}
	return val > 0
}

// MarkApprovalProcessed records an idempotency key as processed.
func (r *RedisClient) MarkApprovalProcessed(ctx context.Context, idempotencyKey string) error {
	key := fmt.Sprintf("approval:idem:%s", idempotencyKey)
	return r.rdb.Set(ctx, key, "processed", idempotencyTTL).Err()
}

// ListPendingApprovals returns all blueprint IDs with PENDING status (for dashboard).
func (r *RedisClient) ListPendingApprovals(ctx context.Context) ([]string, error) {
	pattern := "approval:pending:*"
	keys, err := r.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func approvalKey(blueprintID string) string {
	return fmt.Sprintf("approval:pending:%s", blueprintID)
}

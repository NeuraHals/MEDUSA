package middleware

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Retry executes fn up to maxAttempts times with exponential backoff.
// Returns nil on first success, last error after all attempts exhausted.
func Retry(ctx context.Context, log *zap.Logger, maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
			log.Warn("retry attempt",
				zap.Int("attempt", attempt+1),
				zap.Duration("delay", delay),
				zap.Error(lastErr),
			)
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("all %d attempts failed: %w", maxAttempts, lastErr)
}

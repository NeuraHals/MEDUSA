package middleware

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// CircuitBreaker is a per-target circuit breaker.
type CircuitBreaker struct {
	state            int32
	failures         int32
	failureThreshold int32
	recoveryTimeout  time.Duration
	lastFailure      time.Time
	mu               sync.Mutex
}

const (
	cbClosed   int32 = 0
	cbOpen     int32 = 1
	cbHalfOpen int32 = 2
)

func NewCircuitBreaker(failureThreshold, recoverySecs int) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: int32(failureThreshold),
		recoveryTimeout:  time.Duration(recoverySecs) * time.Second,
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	if atomic.LoadInt32(&cb.state) == cbOpen {
		cb.mu.Lock()
		elapsed := time.Since(cb.lastFailure)
		cb.mu.Unlock()
		if elapsed > cb.recoveryTimeout {
			atomic.StoreInt32(&cb.state, cbHalfOpen)
			return false
		}
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.state, cbClosed)
}

func (cb *CircuitBreaker) RecordFailure() {
	count := atomic.AddInt32(&cb.failures, 1)
	if count >= cb.failureThreshold {
		cb.mu.Lock()
		cb.lastFailure = time.Now()
		cb.mu.Unlock()
		atomic.StoreInt32(&cb.state, cbOpen)
	}
}

// Retry executes fn up to maxAttempts with exponential backoff.
func Retry(ctx context.Context, log *zap.Logger, maxAttempts int, fn func() error) error {
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			delay := time.Duration(200*(1<<uint(i))) * time.Millisecond
			log.Warn("retrying", zap.Int("attempt", i+1), zap.Duration("delay", delay), zap.Error(lastErr))
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

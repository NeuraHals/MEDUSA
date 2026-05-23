package middleware

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the circuit breaker FSM state.
type CircuitState int32

const (
	StateClosed   CircuitState = 0
	StateOpen     CircuitState = 1
	StateHalfOpen CircuitState = 2
)

// CircuitBreaker is a per-provider circuit breaker.
type CircuitBreaker struct {
	state            int32 // atomic CircuitState
	failures         int32
	failureThreshold int32
	recoveryTimeout  time.Duration
	lastFailure      time.Time
	mu               sync.Mutex
}

func NewCircuitBreaker(failureThreshold int, recoverySeconds int) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: int32(failureThreshold),
		recoveryTimeout:  time.Duration(recoverySeconds) * time.Second,
	}
}

// IsOpen returns true if the circuit is open (calls should be blocked).
func (cb *CircuitBreaker) IsOpen() bool {
	state := CircuitState(atomic.LoadInt32(&cb.state))
	if state == StateOpen {
		cb.mu.Lock()
		elapsed := time.Since(cb.lastFailure)
		cb.mu.Unlock()
		if elapsed > cb.recoveryTimeout {
			atomic.StoreInt32(&cb.state, int32(StateHalfOpen))
			return false
		}
		return true
	}
	return false
}

// RecordSuccess resets the circuit to Closed state.
func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.state, int32(StateClosed))
}

// RecordFailure increments failure count and opens the circuit if threshold reached.
func (cb *CircuitBreaker) RecordFailure() {
	count := atomic.AddInt32(&cb.failures, 1)
	if count >= cb.failureThreshold {
		cb.mu.Lock()
		cb.lastFailure = time.Now()
		cb.mu.Unlock()
		atomic.StoreInt32(&cb.state, int32(StateOpen))
	}
}

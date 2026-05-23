use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Mutex;

#[derive(Debug, Clone, PartialEq)]
pub enum CircuitState { Closed, Open, HalfOpen }

pub struct CircuitBreaker {
    state: Arc<Mutex<CircuitState>>,
    failure_count: Arc<AtomicU32>,
    failure_threshold: u32,
    recovery_timeout: Duration,
    last_failure: Arc<Mutex<Option<Instant>>>,
}

impl CircuitBreaker {
    pub fn new(failure_threshold: u32, recovery_timeout_secs: u64) -> Self {
        Self {
            state: Arc::new(Mutex::new(CircuitState::Closed)),
            failure_count: Arc::new(AtomicU32::new(0)),
            failure_threshold,
            recovery_timeout: Duration::from_secs(recovery_timeout_secs),
            last_failure: Arc::new(Mutex::new(None)),
        }
    }

    pub async fn is_open(&self) -> bool {
        let mut state = self.state.lock().await;
        if *state == CircuitState::Open {
            let last = self.last_failure.lock().await;
            if let Some(t) = *last {
                if t.elapsed() > self.recovery_timeout {
                    *state = CircuitState::HalfOpen;
                    return false;
                }
            }
            return true;
        }
        false
    }

    pub async fn record_failure(&self) {
        let count = self.failure_count.fetch_add(1, Ordering::SeqCst) + 1;
        if count >= self.failure_threshold {
            *self.state.lock().await = CircuitState::Open;
            *self.last_failure.lock().await = Some(Instant::now());
        }
    }

    pub async fn record_success(&self) {
        self.failure_count.store(0, Ordering::SeqCst);
        *self.state.lock().await = CircuitState::Closed;
    }
}

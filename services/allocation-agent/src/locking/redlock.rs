use crate::redis::client::RedisClient;
use anyhow::Result;
use rand::Rng;
use std::sync::Arc;
use tracing::debug;

const LOCK_RETRY_COUNT: u32 = 3;
const LOCK_RETRY_DELAY_MS: u64 = 50;

/// RedlockManager implements a simplified Redlock-style distributed locking
/// using Redis SET NX PX (atomic set-if-not-exists with TTL).
pub struct RedlockManager {
    redis: Arc<RedisClient>,
    ttl_ms: u64,
}

impl RedlockManager {
    pub fn new(redis: Arc<RedisClient>, ttl_ms: u64) -> Self {
        Self { redis, ttl_ms }
    }

    /// Attempt to acquire a lock. Returns Ok(true) if lock acquired.
    /// Retries LOCK_RETRY_COUNT times with jittered delays.
    pub async fn acquire(&self, key: &str) -> Result<bool> {
        let token = uuid::Uuid::new_v4().to_string();

        for attempt in 0..LOCK_RETRY_COUNT {
            let acquired = self.redis.set_nx_px(key, &token, self.ttl_ms).await?;
            if acquired {
                debug!(key = %key, attempt, "lock acquired");
                return Ok(true);
            }

            // Jittered exponential backoff
            let jitter = rand::thread_rng().gen_range(0..=20);
            let delay = LOCK_RETRY_DELAY_MS * (2u64.pow(attempt)) + jitter;
            tokio::time::sleep(tokio::time::Duration::from_millis(delay)).await;
        }

        Ok(false)
    }

    /// Release a held lock by deleting the key.
    pub async fn release(&self, key: &str) -> Result<()> {
        self.redis.del(key).await
    }
}

use dashmap::DashMap;
use std::sync::Arc;

/// Local in-memory idempotency cache.
/// Prevents reprocessing a crisis event_id already emitted in the current session.
pub struct IdempotencyCache {
    seen: Arc<DashMap<String, ()>>,
}

impl IdempotencyCache {
    pub fn new() -> Self {
        Self { seen: Arc::new(DashMap::new()) }
    }

    /// Returns true if the key was already seen (duplicate).
    pub fn check_and_insert(&self, key: &str) -> bool {
        if self.seen.contains_key(key) {
            return true;
        }
        self.seen.insert(key.to_string(), ());
        false
    }
}

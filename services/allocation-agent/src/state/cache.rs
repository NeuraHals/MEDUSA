use dashmap::DashMap;
use std::sync::Arc;

pub struct IdempotencyCache {
    seen: Arc<DashMap<String, ()>>,
}

impl IdempotencyCache {
    pub fn new() -> Self {
        Self { seen: Arc::new(DashMap::new()) }
    }

    pub fn check_and_insert(&self, key: &str) -> bool {
        if self.seen.contains_key(key) { return true; }
        self.seen.insert(key.to_string(), ());
        false
    }
}

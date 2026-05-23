use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;

/// Health tracks service readiness state atomically.
#[derive(Clone)]
pub struct Health {
    ready: Arc<AtomicBool>,
}

impl Health {
    pub fn new() -> Self {
        Self { ready: Arc::new(AtomicBool::new(false)) }
    }

    pub fn set_ready(&self, ready: bool) {
        self.ready.store(ready, Ordering::SeqCst);
    }

    pub fn is_ready(&self) -> bool {
        self.ready.load(Ordering::SeqCst)
    }
}

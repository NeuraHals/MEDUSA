use std::sync::atomic::{AtomicBool, Ordering};

pub mod server;

pub struct HealthState {
    ready: AtomicBool,
}

impl HealthState {
    pub fn new() -> Self {
        Self { ready: AtomicBool::new(false) }
    }

    pub fn set_ready(&self, ready: bool) {
        self.ready.store(ready, Ordering::SeqCst);
    }

    pub fn is_ready(&self) -> bool {
        self.ready.load(Ordering::SeqCst)
    }
}

use chrono::{DateTime, Utc};
use dashmap::DashMap;
use std::collections::VecDeque;
use std::sync::Arc;
use tokio::sync::Mutex;

/// A sliding-window event buffer keyed by hospital_id.
/// Retains events within the configured time window for correlation.
pub struct SlidingWindow {
    window_seconds: u64,
    /// hospital_id -> chronological deque of (timestamp, event_id, event_type)
    buckets: Arc<DashMap<String, Mutex<VecDeque<WindowEntry>>>>,
}

#[derive(Clone, Debug)]
pub struct WindowEntry {
    pub timestamp: DateTime<Utc>,
    pub event_id: String,
    pub event_type: String,
    pub source_system: String,
}

impl SlidingWindow {
    pub fn new(window_seconds: u64) -> Self {
        Self {
            window_seconds,
            buckets: Arc::new(DashMap::new()),
        }
    }

    /// Insert an event into the hospital's window, evicting stale entries.
    pub async fn insert(&self, hospital_id: &str, entry: WindowEntry) {
        let cutoff = Utc::now()
            - chrono::Duration::seconds(self.window_seconds as i64);

        let bucket = self
            .buckets
            .entry(hospital_id.to_string())
            .or_insert_with(|| Mutex::new(VecDeque::new()));

        let mut deque = bucket.lock().await;

        // Evict entries outside the window
        while let Some(front) = deque.front() {
            if front.timestamp < cutoff {
                deque.pop_front();
            } else {
                break;
            }
        }
        deque.push_back(entry);
    }

    /// Snapshot the current window contents for a given hospital.
    pub async fn snapshot(&self, hospital_id: &str) -> Vec<WindowEntry> {
        match self.buckets.get(hospital_id) {
            Some(bucket) => bucket.lock().await.iter().cloned().collect(),
            None => vec![],
        }
    }
}

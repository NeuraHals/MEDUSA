use anyhow::Result;
use redis::{aio::ConnectionManager, AsyncCommands, Client};
use serde_json;
use crate::models::{SimulationRecord, SimulationState};
use chrono::Utc;

const IDEMPOTENCY_TTL: u64 = 86_400;    // 24h
const SIM_RECORD_TTL: u64 = 172_800;   // 48h

/// RedisState manages simulation idempotency and state persistence.
#[derive(Clone)]
pub struct RedisState {
    conn: ConnectionManager,
}

impl RedisState {
    pub async fn new(url: &str) -> Result<Self> {
        let client = Client::open(url)?;
        let conn = ConnectionManager::new(client).await?;
        Ok(Self { conn })
    }

    /// Returns true if a simulation request has already been processed.
    /// Fail-safe: returns false on Redis error.
    pub async fn is_duplicate(&self, request_id: &str) -> bool {
        let key = idem_key(request_id);
        let mut c = self.conn.clone();
        match c.exists::<_, i32>(&key).await {
            Ok(v) => v > 0,
            Err(_) => false,
        }
    }

    /// Marks a simulation as processed with 24h TTL.
    pub async fn mark_processed(&self, request_id: &str) -> Result<()> {
        let key = idem_key(request_id);
        let mut c = self.conn.clone();
        c.set_ex::<_, _, ()>(&key, "processed", IDEMPOTENCY_TTL).await?;
        Ok(())
    }

    /// Persists a SimulationRecord with 48h TTL.
    pub async fn store_record(&self, record: &SimulationRecord) -> Result<()> {
        let key = record_key(&record.request_id);
        let data = serde_json::to_string(record)?;
        let mut c = self.conn.clone();
        c.set_ex::<_, _, ()>(&key, &data, SIM_RECORD_TTL).await?;
        Ok(())
    }

    /// Retrieves a SimulationRecord. Returns None if not found.
    pub async fn get_record(&self, request_id: &str) -> Result<Option<SimulationRecord>> {
        let key = record_key(request_id);
        let mut c = self.conn.clone();
        let data: Option<String> = c.get(&key).await?;
        Ok(data.and_then(|d| serde_json::from_str(&d).ok()))
    }

    /// Updates the state field of an existing record.
    pub async fn set_state(&self, request_id: &str, state: SimulationState) -> Result<()> {
        if let Some(mut record) = self.get_record(request_id).await? {
            record.state = state;
            if matches!(record.state, SimulationState::Completed | SimulationState::Failed | SimulationState::Partial) {
                record.completed_at = Some(Utc::now());
            }
            self.store_record(&record).await?;
        }
        Ok(())
    }
}

fn idem_key(request_id: &str) -> String { format!("sipa:idem:{}", request_id) }
fn record_key(request_id: &str) -> String { format!("sipa:sim:{}", request_id) }

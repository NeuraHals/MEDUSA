use anyhow::Result;
use redis::{aio::ConnectionManager, AsyncCommands, Client};
use tracing::info;

/// RedisClient wraps a managed async connection pool to Redis Enterprise.
pub struct RedisClient {
    manager: ConnectionManager,
}

impl RedisClient {
    pub async fn get_str(&self, key: &str) -> Result<Option<String>> {
        let mut conn = self.manager.clone();
        let val: Option<String> = conn.get(key).await?;
        Ok(val)
    }

    pub async fn set_ex(&self, key: &str, value: &str, ttl_secs: u64) -> Result<()> {
        let mut conn = self.manager.clone();
        conn.set_ex::<_, _, ()>(key, value, ttl_secs).await?;
        Ok(())
    }

    pub async fn exists(&self, key: &str) -> Result<bool> {
        let mut conn = self.manager.clone();
        let result: bool = conn.exists(key).await?;
        Ok(result)
    }
}

pub async fn connect(redis_url: &str) -> Result<RedisClient> {
    info!("connecting to Redis at {}", redis_url);
    let client = Client::open(redis_url)?;
    let manager = ConnectionManager::new(client).await?;
    info!("Redis connection established");
    Ok(RedisClient { manager })
}

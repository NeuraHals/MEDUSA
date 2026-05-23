use anyhow::Result;
use redis::{aio::ConnectionManager, AsyncCommands, Client};
use tracing::info;

pub struct RedisClient {
    manager: ConnectionManager,
}

impl RedisClient {
    pub async fn get_str(&self, key: &str) -> Result<Option<String>> {
        let mut c = self.manager.clone();
        Ok(c.get(key).await?)
    }

    pub async fn set_ex(&self, key: &str, value: &str, ttl_secs: u64) -> Result<()> {
        let mut c = self.manager.clone();
        c.set_ex::<_, _, ()>(key, value, ttl_secs).await?;
        Ok(())
    }

    pub async fn exists(&self, key: &str) -> Result<bool> {
        let mut c = self.manager.clone();
        Ok(c.exists(key).await?)
    }

    /// SET key value NX PX ttl_ms — returns true if key was set (lock acquired)
    pub async fn set_nx_px(&self, key: &str, value: &str, ttl_ms: u64) -> Result<bool> {
        let mut c = self.manager.clone();
        let result: Option<String> = redis::cmd("SET")
            .arg(key)
            .arg(value)
            .arg("NX")
            .arg("PX")
            .arg(ttl_ms)
            .query_async(&mut c)
            .await?;
        Ok(result.is_some())
    }

    pub async fn del(&self, key: &str) -> Result<()> {
        let mut c = self.manager.clone();
        let _: () = c.del(key).await?;
        Ok(())
    }

    pub async fn ping(&self) -> Result<()> {
        let mut c = self.manager.clone();
        let _: () = redis::cmd("PING").query_async(&mut c).await?;
        Ok(())
    }
}

pub async fn connect(redis_url: &str) -> Result<RedisClient> {
    info!("connecting to Redis at {}", redis_url);
    let client = Client::open(redis_url)?;
    let manager = ConnectionManager::new(client).await?;
    info!("Redis connected");
    Ok(RedisClient { manager })
}

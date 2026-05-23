use anyhow::Result;
use std::time::Duration;
use tokio::time::sleep;
use tracing::warn;

/// Executes `action` with exponential backoff retry.
/// Max retries: 3. Base delay: 100ms. After 3 failures routes to DLQ.
pub async fn with_retry<F, Fut, T>(action: F, context: &str) -> Result<T>
where
    F: Fn() -> Fut,
    Fut: std::future::Future<Output = Result<T>>,
{
    let mut attempt = 0u32;
    loop {
        match action().await {
            Ok(result) => return Ok(result),
            Err(e) if attempt < 3 => {
                attempt += 1;
                let delay = Duration::from_millis(100 * 2u64.pow(attempt));
                warn!(
                    context = context,
                    attempt = attempt,
                    delay_ms = delay.as_millis(),
                    error = %e,
                    "retrying after error"
                );
                sleep(delay).await;
            }
            Err(e) => return Err(e),
        }
    }
}

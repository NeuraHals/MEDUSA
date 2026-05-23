use tracing_subscriber::{fmt, layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

pub fn init(env: &str) {
    let filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| {
        if env == "production" {
            EnvFilter::new("info")
        } else {
            EnvFilter::new("debug")
        }
    });

    if env == "production" {
        tracing_subscriber::registry()
            .with(filter)
            .with(fmt::layer().json())
            .init();
    } else {
        tracing_subscriber::registry()
            .with(filter)
            .with(fmt::layer().pretty())
            .init();
    }
}

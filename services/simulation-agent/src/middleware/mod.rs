use axum::{
    extract::State,
    response::Json,
    routing::get,
    Router,
};
use serde_json::{json, Value};
use std::sync::Arc;
use crate::health::Health;

#[derive(Clone)]
pub struct AppState {
    pub health: Health,
}

pub fn router(health: Health) -> Router {
    let state = Arc::new(AppState { health });
    Router::new()
        .route("/health/live", get(live))
        .route("/health/ready", get(ready))
        .with_state(state)
}

async fn live() -> Json<Value> {
    Json(json!({ "status": "alive" }))
}

async fn ready(State(state): State<Arc<AppState>>) -> (axum::http::StatusCode, Json<Value>) {
    if state.health.is_ready() {
        (axum::http::StatusCode::OK, Json(json!({ "status": "ready" })))
    } else {
        (
            axum::http::StatusCode::SERVICE_UNAVAILABLE,
            Json(json!({ "status": "not ready" })),
        )
    }
}

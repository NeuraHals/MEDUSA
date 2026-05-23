use crate::health::HealthState;
use axum::{extract::State, http::StatusCode, routing::get, Json, Router};
use serde_json::{json, Value};
use std::sync::Arc;

pub async fn run(addr: String, health: Arc<HealthState>) {
    let app = Router::new()
        .route("/health/live", get(live))
        .route("/health/ready", get(ready))
        .with_state(health);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

async fn live() -> StatusCode { StatusCode::OK }

async fn ready(State(h): State<Arc<HealthState>>) -> (StatusCode, Json<Value>) {
    if h.is_ready() {
        (StatusCode::OK, Json(json!({"status":"ready"})))
    } else {
        (StatusCode::SERVICE_UNAVAILABLE, Json(json!({"status":"not ready"})))
    }
}

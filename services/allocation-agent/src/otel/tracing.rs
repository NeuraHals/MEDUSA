use anyhow::Result;
use opentelemetry::global;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{runtime, trace as sdktrace};

pub fn init_tracer(endpoint: &str, service_name: &str) -> Result<()> {
    let tracer = opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(
            opentelemetry_otlp::new_exporter().tonic().with_endpoint(endpoint),
        )
        .with_trace_config(sdktrace::config().with_resource(
            opentelemetry_sdk::Resource::new(vec![
                opentelemetry::KeyValue::new("service.name", service_name.to_string()),
            ]),
        ))
        .install_batch(runtime::Tokio)?;
    global::set_tracer_provider(tracer.provider().unwrap());
    Ok(())
}

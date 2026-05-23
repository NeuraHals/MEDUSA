fn main() -> Result<(), Box<dyn std::error::Error>> {
    let protoc_path = protoc_bin_vendored::protoc_bin_path()?;
    std::env::set_var("PROTOC", protoc_path);

    // Locate well-known protobuf types bundled alongside the protoc binary
    let protoc_include = std::env::var("PROTOC_INCLUDE").unwrap_or_else(|_| {
        let protoc = std::env::var("PROTOC").unwrap_or_else(|_| "protoc".to_string());
        let protoc_path = std::path::PathBuf::from(protoc);
        protoc_path
            .parent()          // bin/
            .and_then(|p| p.parent()) // root
            .map(|p| p.join("include").to_string_lossy().into_owned())
            .unwrap_or_default()
    });

    let mut config = tonic_build::configure();
    if !protoc_include.is_empty() {
        config = config.protoc_arg(format!("-I{}", protoc_include));
    }

    config.compile(&["proto/allocation.proto"], &["proto/"])?;

    println!("cargo:rerun-if-changed=proto/allocation.proto");
    println!("cargo:rerun-if-changed=proto");
    Ok(())
}

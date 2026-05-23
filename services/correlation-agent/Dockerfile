FROM rust:1.77-alpine AS builder
RUN apk add --no-cache musl-dev openssl-dev pkgconfig cmake make g++

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
# Dummy build to cache dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release && rm -rf src

COPY src ./src
RUN touch src/main.rs && cargo build --release

# --- Distroless final image ---
FROM gcr.io/distroless/cc:nonroot
WORKDIR /
COPY --from=builder /app/target/release/correlation-agent .
COPY --from=builder /app/config.yaml .
USER 65532:65532
EXPOSE 8081
ENTRYPOINT ["/correlation-agent"]

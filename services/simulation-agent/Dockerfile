FROM rust:1.77-alpine AS builder
RUN apk add --no-cache musl-dev cmake make g++ openssl-dev pkgconf librdkafka-dev

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
# Cache dependencies layer
RUN mkdir -p src && echo "fn main(){}" > src/main.rs && cargo build --release --locked && rm -rf src

COPY src ./src
RUN touch src/main.rs && cargo build --release --locked

FROM gcr.io/distroless/cc:nonroot
WORKDIR /
COPY --from=builder /app/target/release/simulation-agent .
COPY --from=builder /app/config.yaml .
USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/simulation-agent"]

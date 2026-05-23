.PHONY: help infra-up infra-down infra-logs proto-gen proto-lint \
        build-go build-rust build-all test-go test-rust test-all \
        docker-build-all ci-validate clean

# ── Configuration ────────────────────────────────────────────────────────────
GO_SERVICES   := ingestion-agent orchestrator-agent oversight-agent \
                 messaging-agent mobile-interaction audit-agent recovery-agent
RUST_SERVICES := correlation-agent confidence-agent allocation-agent simulation-agent

# ── Help ─────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  MEDUSA Monorepo — Available targets:"
	@echo ""
	@echo "  infra-up           Start local dev infrastructure (Kafka, Redis, OTel)"
	@echo "  infra-down         Stop local dev infrastructure"
	@echo "  infra-logs         Tail infrastructure logs"
	@echo "  proto-lint         Lint protobuf definitions with buf"
	@echo "  proto-gen          Generate gRPC stubs from protobuf"
	@echo "  build-go           Build all Go services"
	@echo "  build-rust         Build all Rust services (release)"
	@echo "  build-all          Build all services"
	@echo "  test-go            Run Go test suites"
	@echo "  test-rust          Run Rust test suites"
	@echo "  test-all           Run all tests"
	@echo "  docker-build-all   Build all Docker images"
	@echo "  ci-validate        Full CI validation pass"
	@echo "  clean              Remove build artifacts"
	@echo ""

# ── Infrastructure ────────────────────────────────────────────────────────────
infra-up:
	docker compose up -d
	@echo "Waiting for Kafka to be ready..."
	@sleep 10
	@echo "Infrastructure ready. Jaeger UI: http://localhost:16686 | Kafka UI: http://localhost:8080"

infra-down:
	docker compose down --remove-orphans

infra-logs:
	docker compose logs -f

# ── Protobuf ─────────────────────────────────────────────────────────────────
proto-lint:
	buf lint api/proto

proto-gen:
	buf generate api/proto
	@mkdir -p api/gen/go api/gen/openapi
	@echo "gRPC stubs generated to api/gen/go/"

# ── Go Services ──────────────────────────────────────────────────────────────
build-go:
	@for svc in $(GO_SERVICES); do \
		echo "Building $$svc..."; \
		cd services/$$svc && go build ./... && cd ../..; \
	done

test-go:
	@for svc in $(GO_SERVICES); do \
		echo "Testing $$svc..."; \
		cd services/$$svc && go test -v -race -count=1 ./... && cd ../..; \
	done

tidy-go:
	@for svc in $(GO_SERVICES); do \
		echo "Tidying $$svc..."; \
		cd services/$$svc && go mod tidy && cd ../..; \
	done

vet-go:
	@for svc in $(GO_SERVICES); do \
		echo "Vetting $$svc..."; \
		cd services/$$svc && go vet ./... && cd ../..; \
	done

# ── Rust Services ─────────────────────────────────────────────────────────────
build-rust:
	cargo build --release --workspace

test-rust:
	cargo test --workspace -- --nocapture

clippy-rust:
	cargo clippy --workspace -- -D warnings

# ── Combined ─────────────────────────────────────────────────────────────────
build-all: build-go build-rust

test-all: test-go test-rust

# ── Docker ────────────────────────────────────────────────────────────────────
docker-build-all:
	@for svc in $(GO_SERVICES) $(RUST_SERVICES); do \
		echo "Building Docker image for $$svc..."; \
		docker build -t antigravity/$$svc:latest services/$$svc; \
	done

# ── CI Validation ────────────────────────────────────────────────────────────
ci-validate: proto-lint vet-go clippy-rust test-all
	@echo ""
	@echo "CI validation complete."

# ── Clean ─────────────────────────────────────────────────────────────────────
clean:
	rm -rf api/gen/go/* api/gen/openapi/*
	cargo clean
	@for svc in $(GO_SERVICES); do \
		rm -f services/$$svc/bin/*; \
	done

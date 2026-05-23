# MEDUSA — Healthcare Crisis Intelligence System
# Monorepo Quick-Start Guide

## Prerequisites
- Go 1.22+
- Rust 1.77+ / Cargo
- Docker + Docker Compose
- buf (protobuf toolchain): `brew install bufbuild/buf/buf`

---

## 1. Start Local Infrastructure

```bash
make infra-up
```

Starts: Kafka (port 9093), Redis (6379), OTel Collector (4317), Jaeger UI (16686), Kafka UI (8080)

All 12 Kafka topics are auto-provisioned by `kafka-init`.

---

## 2. Generate gRPC Stubs

```bash
make proto-gen
# or
bash scripts/proto-gen.sh
```

Output: `api/gen/go/` (Go stubs), `api/gen/openapi/` (OpenAPI specs)

---

## 3. Build All Services

```bash
make build-all
```

Builds Go services and the Rust workspace in release mode.

---

## 4. Run Tests

```bash
make test-all
```

---

## 5. Run Full Validation

```bash
bash scripts/validate.sh
```

Validates: structure, module paths, Go build, Rust build, Kafka topic refs, port uniqueness, K8s manifests.

---

## Service Port Map

| Service             | Port  | Protocol |
|---------------------|-------|----------|
| ingestion-agent     | 8080  | HTTP     |
| correlation-agent   | 8081  | HTTP     |
| confidence-agent    | 8082  | HTTP     |
| allocation-agent    | 8083  | HTTP     |
| orchestrator-agent  | 8084  | HTTP     |
| oversight-agent     | 50051 | gRPC     |
| messaging-agent     | 8086  | HTTP     |
| mobile-interaction  | 8087  | HTTP     |
| audit-agent         | 8088  | HTTP     |
| recovery-agent      | 8089  | HTTP     |
| simulation-agent    | 8090  | HTTP     |

---

## Kafka Topic Map

| Topic | Producer | Consumer |
|-------|----------|----------|
| `external.telemetry.v1` | External | SIA |
| `clinical.crisis.event.v1` | SIA | CCA |
| `clinical.orchestration.confidence.v1` | CCA | C&CA |
| `clinical.orchestration.blueprint.v1` | C&CA / RAA | AOA |
| `clinical.orchestration.execution.v1` | AOA | SMA, MIA |
| `clinical.orchestration.approval.v1` | HOA / MIA | AOA |
| `clinical.orchestration.rollback.v1` | AOA | R&AA |
| `clinical.orchestration.recovery.v1` | R&AA | ACA |
| `clinical.orchestration.notification.v1` | SMA | ACA |
| `clinical.simulation.request.v1` | Operator tools | SIPA |
| `clinical.simulation.result.v1` | SIPA | ACA |
| `system.dlq.v1` | All agents | Ops monitoring |

---

## Architecture Freeze
Architecture Freeze v1.0 is active. No new agents or topics may be added without a freeze lift.

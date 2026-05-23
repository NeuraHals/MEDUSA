#!/usr/bin/env bash
# scripts/validate.sh — Full monorepo integration validation script
# Run locally or in CI to assert compilation, lint, and config consistency.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GO_SERVICES=(
  ingestion-agent
  orchestrator-agent
  oversight-agent
  messaging-agent
  mobile-interaction
  audit-agent
  recovery-agent
)

RUST_SERVICES=(
  correlation-agent
  confidence-agent
  allocation-agent
  simulation-agent
)

PASS=0
FAIL=0

log_ok()   { echo "  [OK]  $*"; ((PASS++)) || true; }
log_fail() { echo "  [FAIL] $*"; ((FAIL++)) || true; }

# ── 1. Repository structure validation ──────────────────────────────────────
echo ""
echo "==> [1/8] Repository structure validation"

for svc in "${GO_SERVICES[@]}"; do
  dir="services/$svc"
  [[ -f "$dir/go.mod" ]]     && log_ok "$svc: go.mod"     || log_fail "$svc: go.mod missing"
  [[ -f "$dir/Dockerfile" ]] && log_ok "$svc: Dockerfile"  || log_fail "$svc: Dockerfile missing"
  [[ -f "$dir/config.yaml" ]]&& log_ok "$svc: config.yaml" || log_fail "$svc: config.yaml missing"
  [[ -f "$dir/Makefile" ]]   && log_ok "$svc: Makefile"    || log_fail "$svc: Makefile missing"
  [[ -d "$dir/cmd/server" ]] && log_ok "$svc: cmd/server"  || log_fail "$svc: cmd/server missing"
  [[ -d "$dir/k8s" ]]        && log_ok "$svc: k8s/"        || log_fail "$svc: k8s/ missing"
done

for svc in "${RUST_SERVICES[@]}"; do
  dir="services/$svc"
  [[ -f "$dir/Cargo.toml" ]] && log_ok "$svc: Cargo.toml"  || log_fail "$svc: Cargo.toml missing"
  [[ -f "$dir/Dockerfile" ]] && log_ok "$svc: Dockerfile"   || log_fail "$svc: Dockerfile missing"
  [[ -f "$dir/config.yaml" ]]&& log_ok "$svc: config.yaml"  || log_fail "$svc: config.yaml missing"
  [[ -d "$dir/k8s" ]]        && log_ok "$svc: k8s/"         || log_fail "$svc: k8s/ missing"
done

# ── 2. Protobuf lint ─────────────────────────────────────────────────────────
echo ""
echo "==> [2/8] Protobuf lint (buf)"
if command -v buf &>/dev/null; then
  if buf lint api/proto; then
    log_ok "buf lint passed"
  else
    log_fail "buf lint failed"
  fi
else
  echo "  [SKIP] buf not installed — skipping proto lint"
fi

# ── 3. Go module consistency ─────────────────────────────────────────────────
echo ""
echo "==> [3/8] Go module path consistency"
for svc in "${GO_SERVICES[@]}"; do
  modpath=$(head -1 "services/$svc/go.mod" | awk '{print $2}')
  expected="github.com/antigravity/mono/services/$svc"
  if [[ "$modpath" == "$expected" ]]; then
    log_ok "$svc: module path correct ($modpath)"
  else
    log_fail "$svc: module path mismatch — got '$modpath', want '$expected'"
  fi
done

# ── 4. Go build validation ────────────────────────────────────────────────────
echo ""
echo "==> [4/8] Go build validation"
if command -v go &>/dev/null; then
  for svc in "${GO_SERVICES[@]}"; do
    cd "services/$svc"
    if go build ./... 2>&1; then
      log_ok "$svc: go build passed"
    else
      log_fail "$svc: go build failed"
    fi
    cd "$ROOT"
  done
else
  echo "  [SKIP] go not installed"
fi

# ── 5. Rust workspace build ────────────────────────────────────────────────────
echo ""
echo "==> [5/8] Rust workspace build"
if command -v cargo &>/dev/null; then
  if cargo build --workspace 2>&1; then
    log_ok "Rust workspace build passed"
  else
    log_fail "Rust workspace build failed"
  fi
else
  echo "  [SKIP] cargo not installed"
fi

# ── 6. Kafka topic consistency ─────────────────────────────────────────────────
echo ""
echo "==> [6/8] Kafka topic consistency check"
EXPECTED_TOPICS=(
  "external.telemetry.v1"
  "clinical.crisis.event.v1"
  "clinical.orchestration.blueprint.v1"
  "clinical.orchestration.execution.v1"
  "clinical.orchestration.approval.v1"
  "clinical.orchestration.rollback.v1"
  "clinical.orchestration.recovery.v1"
  "clinical.orchestration.notification.v1"
  "clinical.orchestration.confidence.v1"
  "clinical.simulation.request.v1"
  "clinical.simulation.result.v1"
  "system.dlq.v1"
)
for topic in "${EXPECTED_TOPICS[@]}"; do
  if grep -rl "$topic" services/ --include="*.go" --include="*.rs" --include="*.yaml" &>/dev/null; then
    log_ok "topic referenced: $topic"
  else
    log_fail "topic not referenced anywhere: $topic"
  fi
done

# ── 7. Service port consistency ────────────────────────────────────────────────
echo ""
echo "==> [7/8] Service port uniqueness check"
declare -A PORT_MAP=(
  ["ingestion-agent"]="8080"
  ["correlation-agent"]="8081"
  ["confidence-agent"]="8082"
  ["allocation-agent"]="8083"
  ["orchestrator-agent"]="8084"
  ["oversight-agent"]="50051"
  ["messaging-agent"]="8086"
  ["mobile-interaction"]="8087"
  ["audit-agent"]="8088"
  ["recovery-agent"]="8089"
  ["simulation-agent"]="8090"
)
SEEN_PORTS=()
for svc in "${!PORT_MAP[@]}"; do
  port="${PORT_MAP[$svc]}"
  if [[ " ${SEEN_PORTS[*]} " == *" $port "* ]]; then
    log_fail "port $port used by $svc is already assigned to another service"
  else
    SEEN_PORTS+=("$port")
    log_ok "$svc: port $port unique"
  fi
done

# ── 8. K8s manifest validation ────────────────────────────────────────────────
echo ""
echo "==> [8/8] Kubernetes manifest presence check"
K8S_REQUIRED=(deployment.yaml service.yaml configmap.yaml hpa.yaml networkpolicy.yaml)
ALL_SERVICES=("${GO_SERVICES[@]}" "${RUST_SERVICES[@]}")
for svc in "${ALL_SERVICES[@]}"; do
  for manifest in "${K8S_REQUIRED[@]}"; do
    path="services/$svc/k8s/$manifest"
    [[ -f "$path" ]] && log_ok "$svc/k8s/$manifest" || log_fail "$svc/k8s/$manifest missing"
  done
done

# ── Summary ────────────────────────────────────────────────────────────────────
echo ""
echo "══════════════════════════════════════════════════"
echo "  Validation complete: $PASS passed, $FAIL failed"
echo "══════════════════════════════════════════════════"
if [[ $FAIL -gt 0 ]]; then
  exit 1
fi

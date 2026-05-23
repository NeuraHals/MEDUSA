#!/usr/bin/env bash
# scripts/proto-gen.sh — Generate gRPC stubs from all proto definitions
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> Creating output directories..."
mkdir -p api/gen/go api/gen/openapi

echo "==> Fetching buf dependencies..."
buf dep update api/proto

echo "==> Linting protobuf definitions..."
buf lint api/proto

echo "==> Generating stubs..."
buf generate api/proto

echo ""
echo "Generated files:"
find api/gen -name "*.go" | head -20
echo ""
echo "Proto generation complete."

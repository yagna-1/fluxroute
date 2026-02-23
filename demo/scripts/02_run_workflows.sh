#!/usr/bin/env bash
set -euo pipefail

ROUTER_BASE_URL="${ROUTER_BASE_URL:-http://localhost:8080}"

run_manifest() {
  local manifest_path="$1"
  curl -sS -X POST "$ROUTER_BASE_URL/v1/validate" \
    -H "Content-Type: application/json" \
    -d "{\"manifest_path\":\"$manifest_path\"}" >/dev/null

  curl -sS -X POST "$ROUTER_BASE_URL/v1/run" \
    -H "Content-Type: application/json" \
    -d "{\"manifest_path\":\"$manifest_path\"}" >/dev/null
}

run_manifest "demo/manifests/tenant-a.yaml"
run_manifest "demo/manifests/tenant-b.yaml"

echo "workflows submitted for tenant-a and tenant-b"

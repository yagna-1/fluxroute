#!/usr/bin/env bash
set -euo pipefail

API_KEY="${CONTROLPLANE_API_KEY:-demo-key}"
CP_BASE_URL="${CONTROLPLANE_BASE_URL:-http://localhost:8081}"
ROUTER_BASE_URL="${ROUTER_BASE_URL:-http://localhost:8080}"
MONTH="$(date -u +%Y-%m)"

curl -sS -X POST "$CP_BASE_URL/v1/usage" \
  -H "Content-Type: application/json" \
  -H "X-Role: admin" \
  -H "X-API-Key: $API_KEY" \
  -d '{"tenant_id":"tenant-a","invocations":120}' >/dev/null

curl -sS -X POST "$CP_BASE_URL/v1/usage" \
  -H "Content-Type: application/json" \
  -H "X-Role: admin" \
  -H "X-API-Key: $API_KEY" \
  -d '{"tenant_id":"tenant-b","invocations":87}' >/dev/null

mkdir -p demo/output

curl -sS "$ROUTER_BASE_URL/v1/replay" \
  -H "Content-Type: application/json" \
  -d '{"trace_path":"demo/output/latest-trace.json"}' > demo/output/replay.json

curl -sS "$CP_BASE_URL/v1/billing/summary?month=$MONTH" -H "X-API-Key: $API_KEY" > demo/output/billing-summary.json
curl -sS "$CP_BASE_URL/v1/billing/invoice?tenant_id=tenant-a" -H "X-API-Key: $API_KEY" > demo/output/invoice-tenant-a.json
curl -sS "$CP_BASE_URL/v1/billing/invoice?tenant_id=tenant-b&format=csv" -H "X-API-Key: $API_KEY" > demo/output/invoice-tenant-b.csv

echo "replay + billing artifacts written under demo/output"
ls -1 demo/output | sed 's/^/- /'

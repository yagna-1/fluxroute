#!/usr/bin/env bash
set -euo pipefail

API_KEY="${CONTROLPLANE_API_KEY:-demo-key}"
BASE_URL="${CONTROLPLANE_BASE_URL:-http://localhost:8081}"

create_tenant() {
  local tenant_id="$1"
  curl -sS -X POST "$BASE_URL/v1/tenants" \
    -H "Content-Type: application/json" \
    -H "X-Role: admin" \
    -H "X-API-Key: $API_KEY" \
    -d "{\"id\":\"$tenant_id\"}" >/dev/null || true
}

create_tenant "tenant-a"
create_tenant "tenant-b"

echo "tenants after provisioning:"
curl -sS "$BASE_URL/v1/tenants?page=1&page_size=50" -H "X-API-Key: $API_KEY"
echo

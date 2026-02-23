#!/usr/bin/env bash
set -euo pipefail

ROUTER_BASE_URL="${ROUTER_BASE_URL:-http://localhost:8080}"

set +e
HTTP_CODE=$(curl -sS -o demo/output/failure-response.txt -w "%{http_code}" -X POST "$ROUTER_BASE_URL/v1/run" \
  -H "Content-Type: application/json" \
  -d '{"manifest_path":"demo/manifests/tenant-failure.yaml"}')
set -e

echo "failure scenario submitted, HTTP code: $HTTP_CODE"
if [[ "$HTTP_CODE" != "202" ]]; then
  echo "router returned non-accepted response:" >&2
  cat demo/output/failure-response.txt >&2
fi

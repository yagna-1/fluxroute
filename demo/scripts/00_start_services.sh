#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

mkdir -p demo/output

GO_BIN="${GO_BIN:-$HOME/.local/go1.26.0/bin/go}"
ROUTER_ADDR="${ROUTER_ADDR:-:8080}"
CONTROLPLANE_ADDR="${CONTROLPLANE_ADDR:-:8081}"
API_KEY="${CONTROLPLANE_API_KEY:-demo-key}"

if [[ -f demo/output/router.pid ]] && kill -0 "$(cat demo/output/router.pid)" 2>/dev/null; then
  echo "router already running (pid=$(cat demo/output/router.pid))"
else
  REQUEST_ROLE=admin TRACE_OUTPUT=demo/output/latest-trace.json ROUTER_ADDR="$ROUTER_ADDR" \
    nohup "$GO_BIN" run ./cmd/router serve > demo/output/router.log 2>&1 &
  echo $! > demo/output/router.pid
  echo "started router (pid=$(cat demo/output/router.pid))"
fi

if [[ -f demo/output/controlplane.pid ]] && kill -0 "$(cat demo/output/controlplane.pid)" 2>/dev/null; then
  echo "control plane already running (pid=$(cat demo/output/controlplane.pid))"
else
  CONTROLPLANE_API_KEY="$API_KEY" CONTROLPLANE_ADDR="$CONTROLPLANE_ADDR" \
    nohup "$GO_BIN" run ./cmd/controlplane > demo/output/controlplane.log 2>&1 &
  echo $! > demo/output/controlplane.pid
  echo "started control plane (pid=$(cat demo/output/controlplane.pid))"
fi

for _ in $(seq 1 50); do
  if curl -sf "http://localhost:8080/v1/healthz" >/dev/null && curl -sf "http://localhost:8081/v1/healthz" >/dev/null; then
    echo "router + control plane are healthy"
    exit 0
  fi
  sleep 0.2
done

echo "services did not become healthy in time"
exit 1

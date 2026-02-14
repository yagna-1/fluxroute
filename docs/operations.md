# Operations Guide

## Router

- Start: `make run`
- Version: `go run ./cmd/router --version`
- Metrics: set `METRICS_ENABLED=true`
- Trace export: set `TRACE_OUTPUT=/path/to/trace.json`

## Control Plane

- Start: `make run-controlplane`
- Health: `GET /healthz`, `GET /readyz`
- SLA hook: `GET /sla`
- TLS/mTLS: set `CONTROLPLANE_TLS_ENABLED=true` and cert env vars.

## Coordination modes

- Single-node: `COORDINATION_MODE=memory`
- Shared-host: `COORDINATION_MODE=file`
- Multi-instance: `COORDINATION_MODE=redis` + `COORDINATION_REDIS_URL`

# Operations Guide

## Router runtime

- Manifest mode: `make run MANIFEST_PATH=configs/router.example.yaml`
- API server mode: `make serve`
- Version: `go run ./cmd/router --version`

Router health endpoints in server mode:
- `GET /healthz`
- `GET /readyz`

## CLI workflows

- Validate manifest: `make validate MANIFEST_PATH=...`
- Replay trace: `make replay MANIFEST_PATH=trace.json`
- Scaffold starter pipeline: `make scaffold TARGET_DIR=./generated PIPELINE_NAME=demo`
- Compare two traces: `make debug EXPECTED_TRACE=a.json ACTUAL_TRACE=b.json`

## Control plane

- Start: `make run-controlplane`
- Version: `go run ./cmd/controlplane version`
- Health: `GET /healthz`, `GET /readyz`
- SLA hook: `GET /sla`
- Billing APIs: `/billing/rates`, `/billing/invoice`

## Observability

- Start local stack: `make trace-view`
- Stop local stack: `make trace-down`
- Grafana: `http://localhost:3000` (`admin` / `admin`)
- Prometheus: `http://localhost:9090`
- Jaeger: `http://localhost:16686`

Recommended local env for router telemetry:
- `TRACE_ENABLED=true`
- `TRACE_ENDPOINT=localhost:4317`
- `METRICS_ENABLED=true`
- `METRICS_ADDR=0.0.0.0:2112`
- `CIRCUIT_PROBE_TIMEOUT=5s`

## Linting

- Preferred: `golangci-lint` installed locally, then run `make lint`
- Fallback: `make lint` automatically runs `golangci/golangci-lint` in Docker when local binary is unavailable

## Coordination modes

- Single-node: `COORDINATION_MODE=memory`
- Shared-host: `COORDINATION_MODE=file`
- Multi-instance: `COORDINATION_MODE=redis` with `COORDINATION_REDIS_URL`

## Kubernetes

- Validate manifests: `make k8s-validate`
- Apply manifests: `make k8s-apply`
- Delete manifests: `make k8s-delete`
- Base config: `deploy/k8s/kustomization.yaml`

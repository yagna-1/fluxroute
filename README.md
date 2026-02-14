# FluxRoute Implementation

This directory is a repo-ready implementation built from:
- `../AgentRouter_YC_Report_Full.txt` (end-user/business goals)
- `../AgentRouter_DevTeam_Report.txt` (engineering architecture and contracts)

## Included capabilities

- Deterministic DAG execution with dependency ordering
- Retry engine with linear/exponential/jitter backoff
- Circuit breaker with failure threshold and reset timeout
- Deterministic trace recording, replay, and divergence debugging
- Structured JSON invocation logs and audit log export (JSONL -> CSV)
- OpenTelemetry tracing (OTLP/gRPC)
- In-memory and Prometheus metrics
- RBAC policies for run/validate/replay/admin actions
- Namespace isolation for tenant-aware execution
- Task coordination leases (memory, file, Redis)
- Control-plane service for tenant provisioning, usage metering, rate cards, and invoices
- Provider adapters: OpenAI, Anthropic, Gemini
- SDK package and provider-to-agent helper
- Router HTTP server mode (`serve`) with optional TLS/mTLS
- CLI commands: `run`, `validate`, `replay`, `audit-export`, `scaffold`, `debug`, `version`

## Quick start

```bash
cd <repo-root>
~/.local/go1.26.0/bin/go mod tidy
make test
make build
make build-cli
make build-controlplane
make run
```

If `go` is already on your `PATH`, the `Makefile` will use it automatically.

## Common commands

- `make run MANIFEST_PATH=path/to/manifest.yaml`
- `make serve` (router API server, default `:8080`)
- `make validate MANIFEST_PATH=path/to/manifest.yaml`
- `make replay MANIFEST_PATH=trace.json`
- `make scaffold TARGET_DIR=./generated PIPELINE_NAME=myflow`
- `make debug EXPECTED_TRACE=a.json ACTUAL_TRACE=b.json`
- `make run-controlplane`
- `make bench`
- `make lint` (uses local `golangci-lint` or Docker fallback)
- `make trace-view` / `make trace-down`

## Router API (`cmd/router serve`)

- `GET /healthz`
- `GET /readyz`
- `POST /run` body: `{"manifest_path":"configs/router.example.yaml"}`
- `POST /validate` body: `{"manifest_path":"configs/router.example.yaml"}`
- `POST /replay` body: `{"trace_path":"/path/to/trace.json"}`

## Control-plane API

- `GET /healthz`
- `GET /readyz`
- `GET /sla`
- `POST /tenants` (admin)
- `GET /tenants`
- `POST /usage` (admin)
- `GET /usage?tenant_id=...`
- `GET /billing/rates`
- `POST /billing/rates` (admin)
- `GET /billing/invoice?tenant_id=...`

## Runtime env vars

- Tracing:
  - `TRACE_ENABLED=true`
  - `TRACE_ENDPOINT=localhost:4317`
  - `TRACE_OUTPUT=/tmp/run.trace.json`
- Metrics:
  - `METRICS_ENABLED=true`
  - `METRICS_ADDR=127.0.0.1:2112`
  - `METRICS_TLS_ENABLED=true`
  - `METRICS_TLS_CERT_FILE`, `METRICS_TLS_KEY_FILE`, `METRICS_TLS_CA_FILE`
  - `METRICS_TLS_REQUIRE_CLIENT_CERT=true`
- Security and audit:
  - `REQUEST_ROLE=viewer|operator|admin`
  - `AUDIT_LOG_PATH=/tmp/fluxroute.audit.log`
- Coordination:
  - `COORDINATION_ENABLED=true`
  - `COORDINATION_MODE=file|memory|redis`
  - `COORDINATION_DIR=/tmp/fluxroute-coordination`
  - `COORDINATION_TTL=2m`
  - `COORDINATION_REDIS_URL=redis://localhost:6379/0`
  - `COORDINATION_REDIS_PREFIX=fluxroute`
- Router server TLS:
  - `ROUTER_ADDR=:8080`
  - `ROUTER_TLS_ENABLED=true`
  - `ROUTER_TLS_CERT_FILE`, `ROUTER_TLS_KEY_FILE`, `ROUTER_TLS_CA_FILE`
  - `ROUTER_TLS_REQUIRE_CLIENT_CERT=true`
- Control-plane TLS:
  - `CONTROLPLANE_ADDR=:8081`
  - `CONTROLPLANE_TLS_ENABLED=true`
  - `CONTROLPLANE_TLS_CERT_FILE`, `CONTROLPLANE_TLS_KEY_FILE`, `CONTROLPLANE_TLS_CA_FILE`
  - `CONTROLPLANE_TLS_REQUIRE_CLIENT_CERT=true`

## Deployment assets

- Dockerfiles:
  - `deploy/Dockerfile.router`
  - `deploy/Dockerfile.controlplane`
- Observability stack:
  - `deploy/observability/docker-compose.yml`
  - Prometheus + Grafana + Jaeger + OTel Collector
- Kubernetes manifests:
  - `deploy/k8s/` (`kustomization.yaml` included)
  - `make k8s-validate` / `make k8s-apply` / `make k8s-delete`

## Version output

- `go run ./cmd/router --version`
- `go run ./cmd/cli version`
- `go run ./cmd/controlplane version`

## References

- `docs/end-user-goals.md`
- `docs/development-blueprint.md`
- `docs/goal-to-dev-traceability.md`
- `docs/operations.md`
- `docs/release-checklist.md`

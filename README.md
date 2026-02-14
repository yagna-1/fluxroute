# Agent Router Implementation Seed

This directory is a repo-ready implementation seed built from:
- `../AgentRouter_YC_Report_Full.txt` (end-user/business goals)
- `../AgentRouter_DevTeam_Report.txt` (engineering contracts and implementation guidance)

It is intentionally scaffold-first: structure, contracts, and development guardrails are in place so implementation can start quickly.

## Why this exists

- Separate working area for implementation before promoting to a standalone repository.
- Shared source of truth that connects user-facing goals to engineering milestones.
- Minimal Go project skeleton aligned with the architecture described in the reports.

## What is included

- `docs/end-user-goals.md`: product goals, target users, success criteria.
- `docs/development-blueprint.md`: architecture contracts, runtime design, coding rules.
- `docs/goal-to-dev-traceability.md`: mapping from customer outcomes to build tasks.
- Runtime capabilities:
  - Dependency-aware DAG execution
  - Retry + circuit breaker resilience
  - Deterministic trace recording and replay validation
  - Structured JSON invocation logs
  - In-memory metrics + Prometheus endpoint support
  - OpenTelemetry spans per invocation
  - Provider adapters for OpenAI, Anthropic, and Gemini
  - SDK runtime API and provider-to-agent helper
  - RBAC enforcement and audit logging
  - Namespace isolation and task coordination leases
  - Control-plane service for tenant provisioning and usage metering
- Go project scaffold:
  - `cmd/` router and CLI entrypoints
  - `internal/` runtime internals
  - `pkg/` public API surfaces
  - `tests/` unit, integration, replay placeholders
  - `examples/` starter examples
  - `configs/` manifest template

## Quick start

```bash
cd agent-router-implementation
~/.local/go1.26.0/bin/go mod tidy
make test
make build
make run
make bench
```

If `go` is already on your `PATH`, the `Makefile` will use it automatically.

## Running with a manifest

- Default: `make run` uses `configs/router.example.yaml`
- Custom path: `make run MANIFEST_PATH=path/to/manifest.yaml`
- CLI arg: `go run ./cmd/router path/to/manifest.yaml`

## CLI commands

- `make cli-run` runs via `cmd/cli run`
- `make validate` validates manifest structure and DAG dependencies
- `TRACE_OUTPUT=trace.json make cli-run` records execution trace JSON
- `make replay MANIFEST_PATH=trace.json` replays and compares outputs from a trace
- `run` output now includes JSON invocation logs and metrics summary
- `go run ./cmd/cli run configs/router.example.yaml`
- `go run ./cmd/cli validate configs/router.example.yaml`
- `go run ./cmd/cli replay trace.json`

## Runtime env vars

- `TRACE_ENABLED=true` enables OpenTelemetry tracing
- `TRACE_ENDPOINT=host:4317` sends traces via OTLP/gRPC (if unset, stdout exporter is used)
- `TRACE_OUTPUT=/tmp/run.trace.json` writes deterministic execution trace JSON
- `METRICS_ENABLED=true` enables Prometheus metrics collection
- `METRICS_ADDR=127.0.0.1:2112` metrics HTTP bind address (supports `:0`)
- `METRICS_TLS_ENABLED=true` enables TLS for metrics endpoint
- `METRICS_TLS_CERT_FILE`, `METRICS_TLS_KEY_FILE`, `METRICS_TLS_CA_FILE`
- `METRICS_TLS_REQUIRE_CLIENT_CERT=true` enforces mTLS client cert auth
- `REQUEST_ROLE=viewer|operator|admin` applies RBAC checks for run/validate/replay
- `AUDIT_LOG_PATH=/tmp/agent-router.audit.log` enables JSONL audit trail
- `COORDINATION_ENABLED=true` enables task lease locking
- `COORDINATION_MODE=file|memory`, `COORDINATION_DIR=/tmp/agent-router-coordination`, `COORDINATION_TTL=2m`
- `WORKER_POOL_SIZE`, `CHANNEL_BUFFER`, `DEFAULT_TIMEOUT`
- `CIRCUIT_FAILURE_THRESHOLD`, `CIRCUIT_RESET_TIMEOUT`

## Example manifests

- `examples/simple-pipeline/manifest.yaml`
- `examples/parallel-agents/manifest.yaml`
- `examples/enterprise-workflow/manifest.yaml`

## Next implementation focus

1. Implement enterprise auth/security controls (mTLS, RBAC, audit export).
2. Add benchmark suite against external frameworks.
3. Design distributed coordination for multi-instance router deployments.
4. Add multi-tenant namespace isolation and policy enforcement.

## Control Plane

```bash
make run-controlplane
```

Endpoints:
- `POST /tenants` with body `{\"id\":\"tenant-a\"}` and header `X-Role: admin`
- `GET /tenants`
- `POST /usage` with body `{\"tenant_id\":\"tenant-a\",\"invocations\":10}` and header `X-Role: admin`
- `GET /usage?tenant_id=tenant-a`

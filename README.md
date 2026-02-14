<div align="center">
  <img src="docs/assets/fluxroute-logo.svg" alt="FluxRoute logo" width="900" />

  <p>
    <a href="https://github.com/yagna-1/fluxroute/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/yagna-1/fluxroute/actions/workflows/ci.yml/badge.svg" /></a>
    <img alt="Go" src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" />
    <img alt="License" src="https://img.shields.io/badge/License-Apache%202.0-2ea44f" />
    <img alt="Runtime" src="https://img.shields.io/badge/Runtime-Deterministic-0ea5e9" />
    <img alt="Observability" src="https://img.shields.io/badge/Observability-OTel%20%7C%20Prometheus%20%7C%20Jaeger-f59e0b" />
  </p>

  <p><strong>FluxRoute</strong> is a deterministic, channel-based AI orchestration runtime in Go.</p>
</div>

## Why FluxRoute

- Deterministic DAG execution with replay validation.
- Explicit state, strict manifest validation, and no hidden framework magic.
- Built-in resilience: retry filtering, circuit breaker with half-open probe timeout, and panic containment.
- Production-friendly observability: JSON logs, OpenTelemetry, Prometheus, audit export.
- Enterprise primitives: RBAC, namespace isolation, coordination leases, control-plane APIs.

## Animated identity

<div align="center">
  <img src="docs/assets/fluxroute-logo-animated.svg" alt="FluxRoute animated logo" width="900" />
</div>

## Architecture

```mermaid
flowchart LR
    C[Client / SDK / CLI] --> API[Router API\n/run /validate /replay]
    API --> CFG[Manifest + RBAC + Namespace]
    CFG --> ENG[Router Engine\nDispatcher / Executor / Aggregator]

    ENG --> REG[Agent Registry]
    ENG --> RES[Retry + Circuit Breaker]
    ENG --> TRC[Trace Recorder]
    ENG --> MET[Metrics Recorder]

    TRC --> FILE[Trace JSON]
    TRC --> OTL[OpenTelemetry Export]
    OTL --> JAE[Jaeger via OTel Collector]

    MET --> PROM[Prometheus]
    PROM --> GRAF[Grafana Dashboard]

    CP[Control Plane API] --> TEN[Tenant Provisioning]
    CP --> USE[Usage Metering]
    CP --> BILL[Rate Card + Invoice]

    classDef ingress fill:#14324b,color:#e8f6ff,stroke:#3a6d96,stroke-width:1.5px;
    classDef core fill:#173f2a,color:#e9fff3,stroke:#36b37e,stroke-width:1.5px;
    classDef obs fill:#4a2f10,color:#fff3df,stroke:#ffb454,stroke-width:1.5px;
    classDef ent fill:#3d1634,color:#ffecfb,stroke:#e879f9,stroke-width:1.5px;

    class C,API ingress;
    class CFG,ENG,REG,RES core;
    class TRC,MET,FILE,OTL,JAE,PROM,GRAF obs;
    class CP,TEN,USE,BILL ent;
```

## Execution flow

```mermaid
sequenceDiagram
    autonumber
    participant U as User/Caller
    participant R as Router
    participant M as Manifest Validator
    participant E as Engine
    participant A as Agent(s)
    participant T as Trace
    participant X as Metrics

    U->>R: POST /run {manifest_path}
    R->>M: Load + validate manifest (DAG/RBAC/namespace)
    M-->>R: Validated config
    R->>E: Build execution plan
    E->>A: Dispatch invocations (goroutines + channels)
    A-->>E: AgentOutput / error
    E->>T: Record deterministic steps
    E->>X: Observe invocations/retries/circuit state
    E-->>R: Ordered aggregated result
    R-->>U: Accepted/Result summary
```

## Resilience model

```mermaid
stateDiagram-v2
    [*] --> Closed
    Closed --> Closed: success
    Closed --> Open: failures >= threshold
    Open --> HalfOpen: reset timeout elapsed
    HalfOpen --> Closed: single probe success
    HalfOpen --> Open: single probe failure / probe timeout
    Open --> Open: requests short-circuited
```

## Quick start

```bash
cd <repo-root>
~/.local/go1.26.0/bin/go mod tidy
make test
make lint
make build
make build-cli
make build-controlplane
make run
```

## Core commands

| Goal | Command |
|---|---|
| Run default manifest | `make run` |
| Run custom manifest | `make run MANIFEST_PATH=path/to/manifest.yaml` |
| Start Router API server | `make serve` |
| Validate manifest | `make validate MANIFEST_PATH=path/to/manifest.yaml` |
| Replay deterministic trace | `make replay MANIFEST_PATH=trace.json` |
| Scaffold starter pipeline | `make scaffold TARGET_DIR=./generated PIPELINE_NAME=myflow` |
| Compare expected vs actual traces | `make debug EXPECTED_TRACE=a.json ACTUAL_TRACE=b.json` |
| Start control plane | `make run-controlplane` |
| Benchmark router paths | `make bench` |
| Observability stack up/down | `make trace-view` / `make trace-down` |
| Validate/apply k8s manifests | `make k8s-validate` / `make k8s-apply` |
| Delete k8s manifests | `make k8s-delete` |

## API surface

### Router (`cmd/router serve`)

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/healthz` | Liveness |
| `GET` | `/readyz` | Readiness |
| `POST` | `/run` | Run manifest (`{"manifest_path":"..."}`) |
| `POST` | `/validate` | Validate manifest (`{"manifest_path":"..."}`) |
| `POST` | `/replay` | Replay trace (`{"trace_path":"..."}`) |

### Control plane (`cmd/controlplane`)

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/healthz` | Liveness |
| `GET` | `/readyz` | Readiness |
| `GET` | `/sla` | SLA telemetry snapshot |
| `POST` | `/tenants` | Create tenant (admin) |
| `GET` | `/tenants` | List tenants |
| `POST` | `/usage` | Add usage (admin) |
| `GET` | `/usage?tenant_id=...` | Read tenant usage |
| `GET` | `/billing/rates` | Get pricing |
| `POST` | `/billing/rates` | Update pricing (admin) |
| `GET` | `/billing/invoice?tenant_id=...` | Generate invoice view |

## Observability stack

```mermaid
flowchart TB
    R[FluxRoute Runtime] -->|OTLP gRPC| O[OTel Collector]
    O --> J[Jaeger]
    R -->|/metrics| P[Prometheus]
    P --> G[Grafana]

    classDef a fill:#173f2a,color:#e9fff3,stroke:#36b37e,stroke-width:1.5px;
    classDef b fill:#4a2f10,color:#fff3df,stroke:#ffb454,stroke-width:1.5px;
    class R a;
    class O,J,P,G b;
```

- Local stack definition: `deploy/observability/docker-compose.yml`
- Dashboard JSON: `deploy/observability/grafana/dashboards/fluxroute-overview.json`

## Configuration highlights

- Tracing: `TRACE_ENABLED`, `TRACE_ENDPOINT`, `TRACE_OUTPUT`
- Metrics: `METRICS_ENABLED`, `METRICS_ADDR`, `METRICS_TLS_*`
- Security: `REQUEST_ROLE`, `AUDIT_LOG_PATH`
- Coordination: `COORDINATION_ENABLED`, `COORDINATION_MODE`, `COORDINATION_REDIS_URL`
- Resilience:
  - `CIRCUIT_FAILURE_THRESHOLD`, `CIRCUIT_RESET_TIMEOUT`, `CIRCUIT_PROBE_TIMEOUT`
  - `retry.retryable_errs` behavior via `RetryPolicy.RetryableErrs` in runtime API
- Router TLS: `ROUTER_TLS_ENABLED`, `ROUTER_TLS_*`
- Control-plane TLS: `CONTROLPLANE_TLS_ENABLED`, `CONTROLPLANE_TLS_*`

## Deployment assets

- Docker: `deploy/Dockerfile.router`, `deploy/Dockerfile.controlplane`
- Kubernetes: `deploy/k8s/kustomization.yaml`
- CI workflow: `.github/workflows/ci.yml`

## Version commands

```bash
go run ./cmd/router --version
go run ./cmd/cli version
go run ./cmd/controlplane version
```

## Documentation map

- `docs/end-user-goals.md`
- `docs/development-blueprint.md`
- `docs/goal-to-dev-traceability.md`
- `docs/operations.md`
- `docs/release-checklist.md`
- `docs/requirement-verification-2026-02-14.md`

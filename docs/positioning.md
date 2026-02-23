# Why FluxRoute vs LangGraph / Temporal / Inngest / Prefect

## One-line positioning

FluxRoute is an AI-native orchestration runtime + control plane focused on deterministic replay, tenant isolation, and built-in usage metering/billing primitives.

## Where FluxRoute is strongest

1. Deterministic replay and divergence debugging are first-class:
- Replay from captured traces with deterministic order checks.
- Trace diff tooling to isolate divergence quickly.

2. Multi-tenant control plane primitives are built in:
- Tenant lifecycle endpoints.
- RBAC checks and namespace isolation in runtime semantics.

3. Billing primitives are native to the control plane:
- Usage metering endpoints.
- Monthly aggregation and invoice export (JSON + CSV).

## Competitive interpretation

- LangGraph: excellent agent-graph framework ergonomics, but hosted multi-tenant billing control-plane primitives are not its primary scope.
- Temporal: battle-tested durable orchestration platform, but AI-tenant billing semantics require additional product layers.
- Inngest: strong event workflow developer experience, but deterministic replay + tenant billing bundle is not the default packaged story.
- Prefect: strong orchestration and observability for data/application workflows; tenant billing orchestration still needs product composition.

## Buyer framing (platform/infra teams)

If your problem is "run workflows," many tools fit.
If your problem is "run AI workflows for many tenants with replay-grade determinism and built-in usage-to-invoice path," FluxRoute gives a tighter default stack.

## Source anchors

- FluxRoute implementation: `internal/trace/replay.go`, `internal/trace/debug.go`, `internal/controlplane/service.go`
- LangGraph docs: <https://langchain-ai.github.io/langgraph/>
- Temporal docs (durable execution): <https://docs.temporal.io/workflows>
- Inngest docs (durable steps): <https://www.inngest.com/docs/learn/inngest-steps>
- Prefect docs (work pools/workers): <https://docs.prefect.io/v3/concepts/work-pools>

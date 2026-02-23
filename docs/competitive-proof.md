# Competitive Proof

## Feature comparison (evidence-oriented)

| Capability | FluxRoute | LangGraph | Temporal | Inngest | Prefect |
|---|---|---|---|---|---|
| Deterministic replay built into runtime workflow path | Yes (`/replay`, trace compare) | Graph/state tooling, replay semantics vary by stack | Durable deterministic workflows | Durable step execution | Durable flows/tasks |
| Built-in tenant provisioning API | Yes (`/v1/tenants`) | Not primary scope | Namespace primitives exist, product integration required | Env/apps model; tenant billing composition required | Workspaces/teams; billing composition required |
| Built-in usage metering + invoice endpoints | Yes (`/v1/usage`, `/v1/billing/*`) | Not primary scope | Requires application layer | Requires application layer | Requires application layer |
| Billing summary endpoint (monthly) | Yes (`/v1/billing/summary`) | No native equivalent | No native equivalent | No native equivalent | No native equivalent |

## When NOT to use FluxRoute

- You only need simple single-tenant local workflow execution with no replay/billing requirements.
- Your organization already standardized on another orchestrator and has mature tenant-billing glue layers.
- You require a fully managed hosted SaaS immediately and cannot operate a self-hosted control plane.

## Evidence links

- FluxRoute APIs and code: `internal/controlplane/service.go`, `internal/app/server.go`, `internal/trace/replay.go`
- External benchmark report: `docs/benchmark-external-2026-02-23.md`
- LangGraph docs: <https://langchain-ai.github.io/langgraph/>
- Temporal workflows docs: <https://docs.temporal.io/workflows>
- Inngest steps docs: <https://www.inngest.com/docs/learn/inngest-steps>
- Prefect work pools docs: <https://docs.prefect.io/v3/concepts/work-pools>

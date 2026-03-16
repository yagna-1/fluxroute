# RULES.md - FluxRoute

## Enforced rules (AstraGraph policy: fluxroute-default)

- Every manifest execution must include workflow_id metadata.
- Circuit-breaker behavior must remain enabled for guarded steps.
- Traces must reach a finalized state before export.
- Retry behavior must respect declared workflow limits.

## Human review required (PR, not direct commit)

- Any change to default retry/circuit-breaker thresholds.
- Schema changes to manifest validation.
- Changes that alter replay determinism semantics.

## Auto-blocked (AstraGraph fail-closed)

- Dispatch requests without workflow_id.
- Trace export requests for non-finalized workflows.
- Workflows that disable mandatory guardrails.

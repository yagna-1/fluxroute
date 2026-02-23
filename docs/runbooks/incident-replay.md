# Runbook: Incident Replay

## Purpose

Investigate execution anomalies with deterministic replay and trace divergence tools.

## Steps

1. Capture trace output path:
- Ensure runtime sets `TRACE_OUTPUT=<path>.json`.

2. Replay trace through router API:
```bash
curl -X POST http://localhost:8080/v1/replay \
  -H 'Content-Type: application/json' \
  -d '{"trace_path":"demo/output/latest-trace.json"}'
```

3. Compare expected vs actual traces:
```bash
make debug EXPECTED_TRACE=expected.json ACTUAL_TRACE=actual.json
```

4. Correlate with metrics/logs:
- check retry/circuit metrics
- inspect audit log status for run/validate/replay

## Success criteria

- Replay either confirms determinism or surfaces exact divergence points.

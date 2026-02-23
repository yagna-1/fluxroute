# Start Here

## App developer path (first 30 minutes)

1. Run `make test && make build`.
2. Execute a sample pipeline with `make run`.
3. Use `make replay MANIFEST_PATH=<trace-file>` for deterministic verification.
4. Try scaffold: `make scaffold TARGET_DIR=./generated PIPELINE_NAME=myflow`.

## Platform engineer path

1. Start router API: `make serve`.
2. Start control plane: `make run-controlplane`.
3. Enable telemetry stack: `make trace-view`.
4. Review runbooks:
- `docs/runbooks/tenant-onboarding.md`
- `docs/runbooks/incident-replay.md`
- `docs/runbooks/billing-reconciliation.md`

## Buyer/evaluator path

1. Read `docs/positioning.md`.
2. Run `demo/README.md` end-to-end.
3. Review benchmark and comparison docs:
- `docs/benchmark-baseline-2026-02-14.md`
- `docs/competitive-proof.md`

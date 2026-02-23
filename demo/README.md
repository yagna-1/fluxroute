# FluxRoute Killer Demo: Multi-tenant Billing Sandbox

This demo shows the differentiation path in one flow:

1. Provision isolated tenants in the control plane.
2. Execute deterministic workflows through the router API.
3. Replay traces to prove deterministic outcomes.
4. Meter per-tenant usage and generate billing artifacts.

## Prerequisites

- Go 1.26 (or compatible local setup used in this repo)
- `curl`
- Ports `8080` (router) and `8081` (control plane) available

## Run in under 10 minutes

From repo root:

```bash
./demo/scripts/00_start_services.sh
./demo/scripts/01_provision_tenants.sh
./demo/scripts/02_run_workflows.sh
./demo/scripts/03_replay_and_billing.sh
```

Optional resilience/failure segment:

```bash
./demo/scripts/04_resilience_failure_case.sh
```

Shutdown:

```bash
./demo/scripts/05_stop_services.sh
```

## One-shot script

```bash
./demo/scripts/record-demo.sh
```

## Generated artifacts

- `demo/output/latest-trace.json`
- `demo/output/replay.json`
- `demo/output/billing-summary.json`
- `demo/output/invoice-tenant-a.json`
- `demo/output/invoice-tenant-b.csv`
- `demo/output/router.log`
- `demo/output/controlplane.log`

## Screencast notes

Use `demo/screencast-outline.md` as a 2-4 minute narration script while running `record-demo.sh`.

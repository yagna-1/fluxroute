# FluxRoute Demo Screencast Outline (2-4 min)

## Scene 1: Product framing (20 sec)

- "FluxRoute combines deterministic AI orchestration with multi-tenant metering and billing primitives."
- Show `README.md` and `demo/README.md` quickly.

## Scene 2: Bring up services (30 sec)

- Run `./demo/scripts/00_start_services.sh`.
- Show health checks for `/v1/healthz` on router and control plane.

## Scene 3: Tenant lifecycle + workflow execution (60 sec)

- Run `./demo/scripts/01_provision_tenants.sh`.
- Run `./demo/scripts/02_run_workflows.sh`.
- Point to tenant-specific manifests in `demo/manifests/`.

## Scene 4: Replay + billing (60 sec)

- Run `./demo/scripts/03_replay_and_billing.sh`.
- Open `demo/output/replay.json` and `demo/output/billing-summary.json`.
- Highlight JSON and CSV invoice export.

## Scene 5: Resilience case (30 sec, optional)

- Run `./demo/scripts/04_resilience_failure_case.sh`.
- Explain retry/circuit behavior and deterministic replay support.

## Scene 6: Close (20 sec)

- Show generated assets under `demo/output`.
- Final line: "FluxRoute gives deterministic replay + tenant-aware metering/billing as one cohesive runtime/control-plane stack."

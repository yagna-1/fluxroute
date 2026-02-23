#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

./demo/scripts/00_start_services.sh
./demo/scripts/01_provision_tenants.sh
./demo/scripts/02_run_workflows.sh
sleep 1
./demo/scripts/03_replay_and_billing.sh
./demo/scripts/04_resilience_failure_case.sh

echo "demo sequence complete. run ./demo/scripts/05_stop_services.sh when done."

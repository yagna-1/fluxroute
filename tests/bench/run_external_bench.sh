#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

VENV_DIR="tests/bench/external/.venv"
OUT_DIR="tests/bench/out"
mkdir -p "$OUT_DIR"

if ! python3 -m venv "$VENV_DIR" >/dev/null 2>&1; then
  python3 -m pip install --user --break-system-packages virtualenv >/dev/null
  python3 -m virtualenv "$VENV_DIR" >/dev/null
fi
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"
python -m pip install --upgrade pip
python -m pip install -r tests/bench/external/requirements.txt

TS="$(date -u +%Y%m%dT%H%M%SZ)"
JSON_OUT="$OUT_DIR/external-bench-$TS.json"
CSV_OUT="$OUT_DIR/external-bench-$TS.csv"

python tests/bench/external/bench_external.py \
  --iterations 8 \
  --warmup 2 \
  --samples 3 \
  --frameworks langgraph,prefect,temporal,inngest \
  --out-json "$JSON_OUT" \
  --out-csv "$CSV_OUT"

cp "$JSON_OUT" "$OUT_DIR/external-latest.json"
cp "$CSV_OUT" "$OUT_DIR/external-latest.csv"

echo "external json: $JSON_OUT"
echo "external csv:  $CSV_OUT"

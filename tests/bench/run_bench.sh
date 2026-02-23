#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

OUT_DIR="tests/bench/out"
mkdir -p "$OUT_DIR"
RAW_OUT="$OUT_DIR/bench-$(date -u +%Y%m%dT%H%M%SZ).txt"
CSV_OUT="$OUT_DIR/latest.csv"

~/.local/go1.26.0/bin/go test ./tests/bench/... -run '^$' -bench . -benchmem | tee "$RAW_OUT"

echo "benchmark,ns_per_op,b_per_op,allocs_per_op" > "$CSV_OUT"
awk '/^Benchmark/ {print $1","$3","$5","$7}' "$RAW_OUT" >> "$CSV_OUT"

echo "raw output: $RAW_OUT"
echo "csv output: $CSV_OUT"

#!/usr/bin/env bash
set -euo pipefail

MANIFEST_PATH="${1:-}"
if [[ -n "$MANIFEST_PATH" && ! -f "$MANIFEST_PATH" ]]; then
  echo "manifest not found: $MANIFEST_PATH" >&2
  exit 1
fi

echo "pre-run checks passed"

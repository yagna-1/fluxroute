# Parallel Agents Example

Runs independent stages in the same level and preserves deterministic aggregation.

## Run

```bash
make cli-run MANIFEST_PATH=examples/parallel-agents/manifest.yaml
```

## Record and Replay

```bash
TRACE_OUTPUT=/tmp/parallel.trace.json make cli-run MANIFEST_PATH=examples/parallel-agents/manifest.yaml
make replay MANIFEST_PATH=/tmp/parallel.trace.json
```

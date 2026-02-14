# Enterprise Workflow Example

Multi-stage workflow with retries and trace export enabled.

## Run

```bash
TRACE_OUTPUT=/tmp/enterprise.trace.json make cli-run MANIFEST_PATH=examples/enterprise-workflow/manifest.yaml
```

## Replay

```bash
make replay MANIFEST_PATH=/tmp/enterprise.trace.json
```

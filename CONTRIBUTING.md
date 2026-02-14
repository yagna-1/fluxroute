# Contributing

## Development setup

```bash
make test
make build
make lint
```

## Pull request standards

- Add/adjust tests for every behavior change.
- Keep deterministic behavior guarantees intact.
- Preserve explicit state passing and no hidden runtime state.
- Ensure `make test`, `make lint`, and `make bench` pass.

## Commit conventions

Use conventional prefixes:
- `feat:` new capability
- `fix:` bug fix
- `docs:` documentation changes
- `chore:` tooling/maintenance

# Benchmark Baseline (2026-02-14)

## Methodology

Command:

```bash
go test ./tests/bench/... -run '^$' -bench . -benchmem
```

Environment:
- OS: linux
- Arch: amd64
- CPU: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
- Package: `github.com/your-org/fluxroute/tests/bench`

## Results

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkEngineRunPlan_Sequential10-8` | 104162 | 39480 | 299 |
| `BenchmarkEngineRunPlan_Parallel100-8` | 540178 | 342722 | 1847 |

## Notes

- This is an internal baseline for regression tracking.
- External framework comparison should use workload parity and dedicated harness; see `tests/bench/compare_external.md`.

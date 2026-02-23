# External Benchmark Report (2026-02-23)

## Scope

This run executes workload-parity benchmarks for:
- FluxRoute (Go runtime baseline)
- LangGraph (Python)
- Prefect (Python)

Workload shapes match `tests/bench/router_benchmark_test.go`:
- `sequential10` (10 dependent steps)
- `parallel100` (100 independent steps)

## Commands

```bash
./tests/bench/run_bench.sh
./tests/bench/run_external_bench.sh
```

Artifacts generated under `tests/bench/out/`:
- `latest.csv` (FluxRoute baseline)
- `external-latest.csv`
- `external-latest.json`

## Environment

- Date (UTC): 2026-02-23
- OS: Linux (Ubuntu 24.04)
- CPU: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
- Python: 3.12.3

## Raw results

### FluxRoute (`run_bench.sh`)

| Scenario | ns/op | ms/op | throughput (ops/sec) |
|---|---:|---:|---:|
| sequential10 | 103546 | 0.1035 | 9657.6 |
| parallel100 | 538063 | 0.5381 | 1858.5 |

### External frameworks (`run_external_bench.sh`)

| Framework | Scenario | iterations | throughput (ops/sec) | p50 ms | p95 ms | p99 ms | avg ms | peak RSS MB |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| langgraph | sequential10 | 20 | 428.6174 | 2.3171 | 2.4903 | 2.5150 | 2.3245 | 97.9883 |
| langgraph | parallel100 | 20 | 32.0340 | 27.9820 | 36.7371 | 75.9903 | 31.2050 | 101.1406 |
| prefect | sequential10 | 20 | 0.7186 | 558.7070 | 2698.2546 | 2855.0872 | 1391.6174 | 176.1328 |
| prefect | parallel100 | 20 | 0.1634 | 6185.0759 | 6614.2250 | 6624.8119 | 6119.6687 | 184.7617 |

## Comparative interpretation

Throughput ratio vs FluxRoute baseline (same host, same nominal graph shapes):

| Scenario | FluxRoute vs LangGraph | FluxRoute vs Prefect |
|---|---:|---:|
| sequential10 | 22.53x | 13439.74x |
| parallel100 | 58.02x | 11373.99x |

## Notes and caveats

- This is a cross-runtime comparison (Go vs Python frameworks). Absolute values reflect runtime and framework overhead differences in addition to orchestration design.
- Prefect emits shutdown-time SQLAlchemy cancellation logs in this environment; artifacts are still written and benchmark exit code is successful.
- `run_external_bench.sh` currently benchmarks LangGraph + Prefect directly. Temporal/Inngest are service-backed and should be run in a dedicated infra-backed harness for fair comparison.

## Repro guidance

- Re-run on the same machine class to compare trends.
- Increase `--iterations` in `tests/bench/run_external_bench.sh` for tighter p95/p99 confidence windows.
- Preserve both raw CSV and JSON in CI artifacts for auditability.

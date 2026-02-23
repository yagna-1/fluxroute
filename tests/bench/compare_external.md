# External Benchmark Comparison Guide

This repository includes internal router benchmarks (`make bench`) and a reproducible exporter script (`./tests/bench/run_bench.sh`).

For external framework comparison (LangGraph/Temporal/Inngest/Prefect), capture at minimum:
- Throughput (requests/sec)
- P50/P95/P99 latency
- Peak RSS memory
- CPU utilization

Method:
1. Reuse the same graph shape as `tests/bench/router_benchmark_test.go`.
2. Run warmup and fixed-duration steady-state windows.
3. Pin identical hardware and runtime constraints.
4. Store raw outputs + CSV under `tests/bench/out`.
5. Publish summary under `docs/benchmark-baseline-*.md`.

See `docs/benchmark-baseline-2026-02-14.md` for the current internal baseline.

Automated harness:
- External runner script: `tests/bench/run_external_bench.sh`
- Python harness: `tests/bench/external/bench_external.py`
- Current published external report: `docs/benchmark-external-2026-02-23.md`

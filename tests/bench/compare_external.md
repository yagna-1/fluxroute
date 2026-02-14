# External Benchmark Comparison Guide

This repository includes internal router benchmarks via `make bench`.

To compare with external frameworks (LangGraph/AutoGen), run equivalent workloads in a separate benchmark harness and capture at least:
- Throughput (requests/sec)
- P50/P95/P99 latency
- Peak RSS memory
- CPU utilization

Recommended process:
1. Use the same task graph shape as `tests/bench/router_benchmark_test.go`.
2. Run each framework for warmup + steady-state windows.
3. Store results in CSV and include machine specs.
4. Check in a benchmark report under `docs/` with reproducible commands.

This is intentionally separate because external framework benchmarks require Python runtime + framework-specific harnesses.

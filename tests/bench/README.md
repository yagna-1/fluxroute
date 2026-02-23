# Benchmarks

Run router microbenchmarks:

```bash
make bench
```

Export reproducible raw + CSV artifacts:

```bash
./tests/bench/run_bench.sh
```

Run external-framework benchmark set (LangGraph + Prefect):

```bash
./tests/bench/run_external_bench.sh
```

Current suite:
- `BenchmarkEngineRunPlan_Sequential10`
- `BenchmarkEngineRunPlan_Parallel100`

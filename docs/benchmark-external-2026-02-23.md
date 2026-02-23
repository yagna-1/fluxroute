# External Benchmark Report (2026-02-23)

## Scope

This run executes workload-parity benchmarks for:
- FluxRoute baseline (`run_bench.sh`)
- LangGraph (Python)
- Prefect (Python)
- Temporal (Python SDK local test server)
- Inngest (Python SDK mocked runner)

Scenario set:
- `sequential10` (10 dependent steps)
- `parallel100` (100 independent steps)
- `ai_chain10` (10 dependent remote-call steps against a local HTTP inference stub with 20ms delay)

## Methodology

- External harness: `tests/bench/external/bench_external.py`
- Parameters: `--iterations 8 --warmup 2 --samples 3`
- Startup and cold first-run are split from steady-state metrics.
- Throughput confidence interval is 95% CI over sample means.

Commands:

```bash
./tests/bench/run_bench.sh
./tests/bench/run_external_bench.sh
```

Artifacts (`tests/bench/out/`):
- `latest.csv` (FluxRoute baseline)
- `external-latest.csv`
- `external-latest.json`

## Environment

- External run generated at UTC: `2026-02-23T07:08:38Z`
- OS: Linux (Ubuntu 24.04)
- CPU: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
- Python: 3.12.3

## FluxRoute baseline

| Scenario | ns/op | ms/op | throughput (ops/sec) |
|---|---:|---:|---:|
| sequential10 | 90519 | 0.0905 | 11047.40 |
| parallel100 | 509648 | 0.5096 | 1962.14 |

## External framework aggregates

| Framework | Scenario | startup mean ms | first-run mean ms | throughput mean ops/s | throughput CI95 | p95 mean ms | peak RSS mean MB |
|---|---|---:|---:|---:|---:|---:|---:|
| langgraph | sequential10 | 3.5064 | 3.7327 | 392.2117 | 5.4347 | 2.6590 | 117.8294 |
| langgraph | parallel100 | 24.3531 | 55.5562 | 34.9253 | 5.2193 | 40.9954 | 122.6042 |
| langgraph | ai_chain10 | 4.5221 | 505.5680 | 1.9631 | 0.0077 | 513.8284 | 134.4076 |
| prefect | sequential10 | 0.0017 | 2408.4368 | 0.6386 | 0.1982 | 2648.6266 | 194.1029 |
| prefect | parallel100 | 0.0043 | 6046.5058 | 0.1630 | 0.0037 | 6469.2146 | 202.6029 |
| prefect | ai_chain10 | 0.0053 | 1782.4446 | 0.8614 | 0.1362 | 1918.7324 | 207.3646 |
| temporal | sequential10 | 208.3539 | 128.6147 | 0.9525 | 0.0000 | 1052.8788 | 221.9232 |
| temporal | parallel100 | 105.4888 | 1589.5040 | 0.5820 | 0.1051 | 2274.6252 | 224.0143 |
| temporal | ai_chain10 | 105.1968 | 629.6240 | 0.9088 | 0.0007 | 1107.2657 | 234.7930 |
| inngest | sequential10 | 54.8210 | 8.4269 | 267.6142 | 0.9197 | 3.8776 | 236.0820 |
| inngest | parallel100 | 54.7611 | 177.7482 | 5.7463 | 0.0476 | 175.9940 | 236.0820 |
| inngest | ai_chain10 | 57.9831 | 525.9637 | 1.9294 | 0.0449 | 527.4112 | 242.8398 |

## Throughput ratio vs FluxRoute baseline

Same host, same nominal scenario shape:

| Scenario | FluxRoute vs LangGraph | FluxRoute vs Inngest | FluxRoute vs Temporal | FluxRoute vs Prefect |
|---|---:|---:|---:|---:|
| sequential10 | 28.17x | 41.28x | 11598.24x | 17299.92x |
| parallel100 | 56.18x | 341.46x | 3371.43x | 12041.01x |

## AI-chain interpretation (`ai_chain10`)

Throughput ranking for this local stubbed model-call workload:
1. LangGraph: `1.9631 ops/s`
2. Inngest: `1.9294 ops/s`
3. Temporal: `0.9088 ops/s`
4. Prefect: `0.8614 ops/s`

## Notes and caveats

- This is a cross-runtime comparison (Go baseline vs Python frameworks). Absolute numbers include runtime-level overhead, not only orchestration semantics.
- Temporal measurements include local test-server + worker process costs in harness startup/first-run windows.
- Inngest measurements use the official Python mocked runner (no remote control plane/network).
- Prefect startup in this harness is lazy, so first-run costs dominate.
- Preserve raw JSON + CSV artifacts for auditability and recomputation.

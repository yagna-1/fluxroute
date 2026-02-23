#!/usr/bin/env python3
from __future__ import annotations

import argparse
import csv
import json
import math
import os
import statistics
import time
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Annotated, Callable, TypedDict

import operator
import psutil

os.environ.setdefault("PREFECT_LOGGING_LEVEL", "ERROR")
os.environ.setdefault("PREFECT_UI_ENABLED", "0")

from langgraph.graph import END, START, StateGraph
from prefect import flow, task
from prefect.task_runners import ThreadPoolTaskRunner


class SeqState(TypedDict):
    value: int


class ParallelState(TypedDict):
    hits: Annotated[list[int], operator.add]


@dataclass
class Result:
    framework: str
    scenario: str
    iterations: int
    throughput_ops_sec: float
    p50_ms: float
    p95_ms: float
    p99_ms: float
    avg_ms: float
    peak_rss_mb: float
    cpu_util_pct: float


def percentile(values: list[float], p: float) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    idx = (len(ordered) - 1) * p
    lo = math.floor(idx)
    hi = math.ceil(idx)
    if lo == hi:
        return ordered[lo]
    frac = idx - lo
    return ordered[lo] * (1.0 - frac) + ordered[hi] * frac


def benchmark_run(name: str, iterations: int, warmup: int, run_once: Callable[[], object]) -> Result:
    for _ in range(warmup):
        run_once()

    process = psutil.Process(os.getpid())
    cpu_start = process.cpu_times()
    wall_start = time.perf_counter()
    peak_rss = process.memory_info().rss
    latencies: list[float] = []

    for i in range(iterations):
        t0 = time.perf_counter()
        run_once()
        latencies.append((time.perf_counter() - t0) * 1000.0)
        if i % 10 == 0:
            peak_rss = max(peak_rss, process.memory_info().rss)

    wall_total = time.perf_counter() - wall_start
    cpu_end = process.cpu_times()
    cpu_total = (cpu_end.user + cpu_end.system) - (cpu_start.user + cpu_start.system)
    cpu_util = (cpu_total / wall_total) * 100.0 if wall_total > 0 else 0.0

    framework, scenario = name.split(":", 1)
    return Result(
        framework=framework,
        scenario=scenario,
        iterations=iterations,
        throughput_ops_sec=iterations / wall_total,
        p50_ms=percentile(latencies, 0.50),
        p95_ms=percentile(latencies, 0.95),
        p99_ms=percentile(latencies, 0.99),
        avg_ms=statistics.mean(latencies),
        peak_rss_mb=peak_rss / (1024.0 * 1024.0),
        cpu_util_pct=cpu_util,
    )


def build_langgraph_seq10() -> Callable[[], object]:
    graph = StateGraph(SeqState)
    previous = START

    for i in range(10):
        node_name = f"n{i+1}"

        def make_node() -> Callable[[SeqState], SeqState]:
            return lambda state: {"value": state["value"] + 1}

        graph.add_node(node_name, make_node())
        graph.add_edge(previous, node_name)
        previous = node_name

    graph.add_edge(previous, END)
    app = graph.compile()

    def run() -> object:
        return app.invoke({"value": 0})

    return run


def build_langgraph_parallel100() -> Callable[[], object]:
    graph = StateGraph(ParallelState)

    for i in range(100):
        node_name = f"p{i+1}"

        def make_node() -> Callable[[ParallelState], ParallelState]:
            return lambda _state: {"hits": [1]}

        graph.add_node(node_name, make_node())
        graph.add_edge(START, node_name)
        graph.add_edge(node_name, END)

    app = graph.compile()

    def run() -> object:
        return app.invoke({"hits": []})

    return run


@task
def increment(x: int) -> int:
    return x + 1


@task
def one() -> int:
    return 1


@flow(log_prints=False)
def prefect_seq10_flow() -> int:
    value = 0
    for _ in range(10):
        value = increment(value)
    return value


@flow(task_runner=ThreadPoolTaskRunner(max_workers=128), log_prints=False)
def prefect_parallel100_flow() -> int:
    futures = [one.submit() for _ in range(100)]
    return sum(f.result() for f in futures)


def build_prefect_seq10() -> Callable[[], object]:
    return lambda: prefect_seq10_flow()


def build_prefect_parallel100() -> Callable[[], object]:
    return lambda: prefect_parallel100_flow()


def main() -> int:
    parser = argparse.ArgumentParser(description="Run external orchestration benchmark set.")
    parser.add_argument("--iterations", type=int, default=120)
    parser.add_argument("--warmup", type=int, default=8)
    parser.add_argument("--out-json", type=Path, required=True)
    parser.add_argument("--out-csv", type=Path, required=True)
    args = parser.parse_args()

    runners: list[tuple[str, Callable[[], object]]] = [
        ("langgraph:sequential10", build_langgraph_seq10()),
        ("langgraph:parallel100", build_langgraph_parallel100()),
        ("prefect:sequential10", build_prefect_seq10()),
        ("prefect:parallel100", build_prefect_parallel100()),
    ]

    results: list[Result] = []
    for name, runner in runners:
        print(f"running {name}...", flush=True)
        results.append(benchmark_run(name=name, iterations=args.iterations, warmup=args.warmup, run_once=runner))

    args.out_json.parent.mkdir(parents=True, exist_ok=True)
    args.out_csv.parent.mkdir(parents=True, exist_ok=True)

    payload = {
        "generated_at_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "python": os.sys.version,
        "results": [asdict(r) for r in results],
    }
    args.out_json.write_text(json.dumps(payload, indent=2), encoding="utf-8")

    with args.out_csv.open("w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            "framework",
            "scenario",
            "iterations",
            "throughput_ops_sec",
            "p50_ms",
            "p95_ms",
            "p99_ms",
            "avg_ms",
            "peak_rss_mb",
            "cpu_util_pct",
        ])
        for r in results:
            writer.writerow([
                r.framework,
                r.scenario,
                r.iterations,
                f"{r.throughput_ops_sec:.4f}",
                f"{r.p50_ms:.4f}",
                f"{r.p95_ms:.4f}",
                f"{r.p99_ms:.4f}",
                f"{r.avg_ms:.4f}",
                f"{r.peak_rss_mb:.4f}",
                f"{r.cpu_util_pct:.2f}",
            ])

    print(f"wrote: {args.out_json}")
    print(f"wrote: {args.out_csv}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

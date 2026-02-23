#!/usr/bin/env python3
from __future__ import annotations

import argparse
import asyncio
import csv
import json
import math
import os
import socket
import statistics
import threading
import time
import uuid
from dataclasses import asdict, dataclass
from datetime import timedelta
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Annotated, Callable, TypedDict

import operator
import psutil

os.environ.setdefault("PREFECT_LOGGING_LEVEL", "ERROR")
os.environ.setdefault("PREFECT_UI_ENABLED", "0")

import httpx
import inngest
from inngest.experimental.mocked.client import Inngest as MockInngest
from inngest.experimental.mocked.trigger import trigger as inngest_trigger
from langgraph.graph import END, START, StateGraph
from prefect import flow, task
from prefect.task_runners import ThreadPoolTaskRunner
from temporalio import activity, workflow
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import UnsandboxedWorkflowRunner, Worker


class SeqState(TypedDict):
    value: int


class ParallelState(TypedDict):
    hits: Annotated[list[int], operator.add]


class _AIHandler(BaseHTTPRequestHandler):
    delay_ms = 20

    def do_GET(self) -> None:  # noqa: N802
        time.sleep(self.delay_ms / 1000.0)
        payload = b'{"ok":true}'
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, format: str, *args: object) -> None:  # noqa: A003
        return


class AIServer:
    def __init__(self, delay_ms: int = 20) -> None:
        self.delay_ms = delay_ms
        self._thread: threading.Thread | None = None
        self._server: ThreadingHTTPServer | None = None
        self.url = ""

    def __enter__(self) -> "AIServer":
        sock = socket.socket()
        sock.bind(("127.0.0.1", 0))
        host, port = sock.getsockname()
        sock.close()

        _AIHandler.delay_ms = self.delay_ms
        self._server = ThreadingHTTPServer((host, port), _AIHandler)
        self.url = f"http://{host}:{port}/inference"
        self._thread = threading.Thread(target=self._server.serve_forever, daemon=True)
        self._thread.start()
        return self

    def __exit__(self, exc_type: object, exc: object, tb: object) -> None:
        if self._server:
            self._server.shutdown()
            self._server.server_close()
        if self._thread:
            self._thread.join(timeout=2)


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


@dataclass
class SampleResult:
    framework: str
    scenario: str
    sample_idx: int
    iterations: int
    startup_ms: float
    first_run_ms: float
    throughput_ops_sec: float
    p50_ms: float
    p95_ms: float
    p99_ms: float
    avg_ms: float
    peak_rss_mb: float
    cpu_util_pct: float


@dataclass
class AggregateResult:
    framework: str
    scenario: str
    samples: int
    iterations_per_sample: int
    startup_ms_mean: float
    startup_ms_p95: float
    first_run_ms_mean: float
    throughput_mean: float
    throughput_ci95: float
    p50_mean: float
    p95_mean: float
    p99_mean: float
    avg_ms_mean: float
    peak_rss_mb_mean: float
    cpu_util_pct_mean: float


RunnerFactory = Callable[[str, str], tuple[Callable[[], object], Callable[[], None]]]


def _measure_sample(
    framework: str,
    scenario: str,
    sample_idx: int,
    iterations: int,
    warmup: int,
    factory: RunnerFactory,
    ai_url: str,
) -> SampleResult:
    process = psutil.Process(os.getpid())

    startup_t0 = time.perf_counter()
    run_once, cleanup = factory(scenario, ai_url)
    startup_ms = (time.perf_counter() - startup_t0) * 1000.0

    try:
        t0 = time.perf_counter()
        run_once()
        first_run_ms = (time.perf_counter() - t0) * 1000.0

        for _ in range(warmup):
            run_once()

        cpu_start = process.cpu_times()
        wall_start = time.perf_counter()
        peak_rss = process.memory_info().rss

        latencies: list[float] = []
        for i in range(iterations):
            t0 = time.perf_counter()
            run_once()
            latencies.append((time.perf_counter() - t0) * 1000.0)
            if i % 5 == 0:
                peak_rss = max(peak_rss, process.memory_info().rss)

        wall_total = time.perf_counter() - wall_start
        cpu_end = process.cpu_times()
        cpu_total = (cpu_end.user + cpu_end.system) - (cpu_start.user + cpu_start.system)

        return SampleResult(
            framework=framework,
            scenario=scenario,
            sample_idx=sample_idx,
            iterations=iterations,
            startup_ms=startup_ms,
            first_run_ms=first_run_ms,
            throughput_ops_sec=iterations / wall_total if wall_total > 0 else 0.0,
            p50_ms=percentile(latencies, 0.50),
            p95_ms=percentile(latencies, 0.95),
            p99_ms=percentile(latencies, 0.99),
            avg_ms=statistics.mean(latencies),
            peak_rss_mb=peak_rss / (1024.0 * 1024.0),
            cpu_util_pct=(cpu_total / wall_total) * 100.0 if wall_total > 0 else 0.0,
        )
    finally:
        cleanup()


def _aggregate(samples: list[SampleResult]) -> AggregateResult:
    framework = samples[0].framework
    scenario = samples[0].scenario
    n = len(samples)

    throughput = [s.throughput_ops_sec for s in samples]
    t_mean = statistics.mean(throughput)
    t_std = statistics.stdev(throughput) if n > 1 else 0.0
    t_ci95 = 1.96 * t_std / math.sqrt(n) if n > 1 else 0.0

    return AggregateResult(
        framework=framework,
        scenario=scenario,
        samples=n,
        iterations_per_sample=samples[0].iterations,
        startup_ms_mean=statistics.mean([s.startup_ms for s in samples]),
        startup_ms_p95=percentile([s.startup_ms for s in samples], 0.95),
        first_run_ms_mean=statistics.mean([s.first_run_ms for s in samples]),
        throughput_mean=t_mean,
        throughput_ci95=t_ci95,
        p50_mean=statistics.mean([s.p50_ms for s in samples]),
        p95_mean=statistics.mean([s.p95_ms for s in samples]),
        p99_mean=statistics.mean([s.p99_ms for s in samples]),
        avg_ms_mean=statistics.mean([s.avg_ms for s in samples]),
        peak_rss_mb_mean=statistics.mean([s.peak_rss_mb for s in samples]),
        cpu_util_pct_mean=statistics.mean([s.cpu_util_pct for s in samples]),
    )


# LangGraph

def build_langgraph_runner(scenario: str, ai_url: str) -> tuple[Callable[[], object], Callable[[], None]]:
    if scenario == "sequential10":
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
        return lambda: app.invoke({"value": 0}), lambda: None

    if scenario == "parallel100":
        graph = StateGraph(ParallelState)
        for i in range(100):
            node_name = f"p{i+1}"

            def make_node() -> Callable[[ParallelState], ParallelState]:
                return lambda _state: {"hits": [1]}

            graph.add_node(node_name, make_node())
            graph.add_edge(START, node_name)
            graph.add_edge(node_name, END)
        app = graph.compile()
        return lambda: app.invoke({"hits": []}), lambda: None

    if scenario == "ai_chain10":
        graph = StateGraph(SeqState)
        previous = START

        for i in range(10):
            node_name = f"ai{i+1}"

            def make_node() -> Callable[[SeqState], SeqState]:
                def _node(state: SeqState) -> SeqState:
                    _ = httpx.get(ai_url, timeout=5.0)
                    return {"value": state["value"] + 1}

                return _node

            graph.add_node(node_name, make_node())
            graph.add_edge(previous, node_name)
            previous = node_name

        graph.add_edge(previous, END)
        app = graph.compile()
        return lambda: app.invoke({"value": 0}), lambda: None

    raise ValueError(f"unknown scenario: {scenario}")


# Prefect

@task
def _pref_inc(x: int) -> int:
    return x + 1


@task
def _pref_one() -> int:
    return 1


@task
def _pref_call_ai(ai_url: str) -> int:
    _ = httpx.get(ai_url, timeout=5.0)
    return 1


@flow(log_prints=False)
def _pref_seq10_flow() -> int:
    value = 0
    for _ in range(10):
        value = _pref_inc(value)
    return value


@flow(task_runner=ThreadPoolTaskRunner(max_workers=128), log_prints=False)
def _pref_parallel100_flow() -> int:
    futures = [_pref_one.submit() for _ in range(100)]
    return sum(f.result() for f in futures)


@flow(log_prints=False)
def _pref_ai_chain10_flow(ai_url: str) -> int:
    value = 0
    for _ in range(10):
        value += _pref_call_ai(ai_url)
    return value


def build_prefect_runner(scenario: str, ai_url: str) -> tuple[Callable[[], object], Callable[[], None]]:
    if scenario == "sequential10":
        return lambda: _pref_seq10_flow(), lambda: None
    if scenario == "parallel100":
        return lambda: _pref_parallel100_flow(), lambda: None
    if scenario == "ai_chain10":
        return lambda: _pref_ai_chain10_flow(ai_url), lambda: None
    raise ValueError(f"unknown scenario: {scenario}")


# Temporal

@activity.defn
async def _t_inc(x: int) -> int:
    return x + 1


@activity.defn
async def _t_one() -> int:
    return 1


@activity.defn
async def _t_call_ai(ai_url: str) -> int:
    _ = httpx.get(ai_url, timeout=5.0)
    return 1


@workflow.defn
class _TemporalSeq10Workflow:
    @workflow.run
    async def run(self, value: int) -> int:
        for _ in range(10):
            value = await workflow.execute_activity(_t_inc, value, start_to_close_timeout=timedelta(seconds=10))
        return value


@workflow.defn
class _TemporalParallel100Workflow:
    @workflow.run
    async def run(self, _seed: int) -> int:
        futures = [
            workflow.execute_activity(_t_one, start_to_close_timeout=timedelta(seconds=10)) for _ in range(100)
        ]
        vals = await asyncio.gather(*futures)
        return sum(vals)


@workflow.defn
class _TemporalAIChain10Workflow:
    @workflow.run
    async def run(self, ai_url: str) -> int:
        value = 0
        for _ in range(10):
            value += await workflow.execute_activity(
                _t_call_ai,
                ai_url,
                start_to_close_timeout=timedelta(seconds=10),
            )
        return value


class TemporalRunner:
    def __init__(self, scenario: str, ai_url: str) -> None:
        self.scenario = scenario
        self.ai_url = ai_url
        self.loop = asyncio.new_event_loop()
        self.env: WorkflowEnvironment | None = None
        self.worker: Worker | None = None
        self.task_queue = f"bench-{uuid.uuid4().hex}"
        self._setup()

    async def _setup_async(self) -> None:
        self.env = await WorkflowEnvironment.start_local()

        workflows = {
            "sequential10": _TemporalSeq10Workflow,
            "parallel100": _TemporalParallel100Workflow,
            "ai_chain10": _TemporalAIChain10Workflow,
        }
        wf_cls = workflows[self.scenario]
        self.worker = Worker(
            self.env.client,
            task_queue=self.task_queue,
            workflows=[wf_cls],
            activities=[_t_inc, _t_one, _t_call_ai],
            workflow_runner=UnsandboxedWorkflowRunner(),
        )
        await self.worker.__aenter__()

    def _setup(self) -> None:
        self.loop.run_until_complete(self._setup_async())

    async def _run_once_async(self) -> object:
        assert self.env is not None
        wf_id = f"wf-{uuid.uuid4().hex}"
        if self.scenario == "sequential10":
            return await self.env.client.execute_workflow(
                _TemporalSeq10Workflow.run,
                0,
                id=wf_id,
                task_queue=self.task_queue,
            )
        if self.scenario == "parallel100":
            return await self.env.client.execute_workflow(
                _TemporalParallel100Workflow.run,
                0,
                id=wf_id,
                task_queue=self.task_queue,
            )
        return await self.env.client.execute_workflow(
            _TemporalAIChain10Workflow.run,
            self.ai_url,
            id=wf_id,
            task_queue=self.task_queue,
        )

    def run_once(self) -> object:
        return self.loop.run_until_complete(self._run_once_async())

    async def _shutdown_async(self) -> None:
        if self.worker is not None:
            await self.worker.__aexit__(None, None, None)
        if self.env is not None:
            await self.env.shutdown()

    def close(self) -> None:
        self.loop.run_until_complete(self._shutdown_async())
        self.loop.close()


def build_temporal_runner(scenario: str, ai_url: str) -> tuple[Callable[[], object], Callable[[], None]]:
    tr = TemporalRunner(scenario, ai_url)
    return tr.run_once, tr.close


# Inngest (mocked test runner from official SDK)


def build_inngest_runner(scenario: str, ai_url: str) -> tuple[Callable[[], object], Callable[[], None]]:
    client = MockInngest(app_id="bench")

    if scenario == "sequential10":

        @client.create_function(fn_id="bench-seq10", trigger=inngest.TriggerEvent(event="bench/seq10"))
        def fn(ctx: inngest.ContextSync) -> dict[str, int]:
            value = 0
            for i in range(10):
                value = ctx.step.run(f"inc-{i}", lambda x=value: x + 1)
            return {"value": value}

        def run_once() -> object:
            ev = inngest.Event(name="bench/seq10", data={"n": 1})
            res = inngest_trigger(fn, ev, client)
            return res.output

        return run_once, lambda: None

    if scenario == "parallel100":

        @client.create_function(fn_id="bench-par100", trigger=inngest.TriggerEvent(event="bench/par100"))
        def fn(ctx: inngest.ContextSync) -> dict[str, int]:
            total = 0
            for i in range(100):
                total += ctx.step.run(f"hit-{i}", lambda: 1)
            return {"total": total}

        def run_once() -> object:
            ev = inngest.Event(name="bench/par100", data={"n": 1})
            res = inngest_trigger(fn, ev, client)
            return res.output

        return run_once, lambda: None

    if scenario == "ai_chain10":

        @client.create_function(fn_id="bench-ai10", trigger=inngest.TriggerEvent(event="bench/ai10"))
        def fn(ctx: inngest.ContextSync) -> dict[str, int]:
            value = 0
            for i in range(10):
                def _call() -> int:
                    _ = httpx.get(ai_url, timeout=5.0)
                    return 1

                value += ctx.step.run(f"ai-{i}", _call)
            return {"value": value}

        def run_once() -> object:
            ev = inngest.Event(name="bench/ai10", data={"n": 1})
            res = inngest_trigger(fn, ev, client)
            return res.output

        return run_once, lambda: None

    raise ValueError(f"unknown scenario: {scenario}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Run external orchestration benchmark set.")
    parser.add_argument("--iterations", type=int, default=8)
    parser.add_argument("--warmup", type=int, default=2)
    parser.add_argument("--samples", type=int, default=3)
    parser.add_argument(
        "--frameworks",
        type=str,
        default="langgraph,prefect,temporal,inngest",
        help="comma-separated frameworks: langgraph,prefect,temporal,inngest",
    )
    parser.add_argument("--out-json", type=Path, required=True)
    parser.add_argument("--out-csv", type=Path, required=True)
    args = parser.parse_args()

    framework_map: dict[str, RunnerFactory] = {
        "langgraph": build_langgraph_runner,
        "prefect": build_prefect_runner,
        "temporal": build_temporal_runner,
        "inngest": build_inngest_runner,
    }

    selected = [f.strip().lower() for f in args.frameworks.split(",") if f.strip()]
    for fw in selected:
        if fw not in framework_map:
            raise ValueError(f"unknown framework in --frameworks: {fw}")

    scenarios = ["sequential10", "parallel100", "ai_chain10"]

    all_samples: list[SampleResult] = []

    with AIServer(delay_ms=20) as ai:
        for fw in selected:
            factory = framework_map[fw]
            for scenario in scenarios:
                print(f"running {fw}:{scenario} ({args.samples} samples)...", flush=True)
                for i in range(args.samples):
                    sample = _measure_sample(
                        framework=fw,
                        scenario=scenario,
                        sample_idx=i + 1,
                        iterations=args.iterations,
                        warmup=args.warmup,
                        factory=factory,
                        ai_url=ai.url,
                    )
                    all_samples.append(sample)

    grouped: dict[tuple[str, str], list[SampleResult]] = {}
    for s in all_samples:
        grouped.setdefault((s.framework, s.scenario), []).append(s)

    aggregates = [_aggregate(grouped[key]) for key in sorted(grouped.keys())]

    args.out_json.parent.mkdir(parents=True, exist_ok=True)
    args.out_csv.parent.mkdir(parents=True, exist_ok=True)

    payload = {
        "generated_at_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "python": os.sys.version,
        "iterations": args.iterations,
        "warmup": args.warmup,
        "samples": args.samples,
        "frameworks": selected,
        "scenarios": scenarios,
        "aggregates": [asdict(a) for a in aggregates],
        "sample_results": [asdict(s) for s in all_samples],
    }
    args.out_json.write_text(json.dumps(payload, indent=2), encoding="utf-8")

    with args.out_csv.open("w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            "framework",
            "scenario",
            "samples",
            "iterations_per_sample",
            "startup_ms_mean",
            "startup_ms_p95",
            "first_run_ms_mean",
            "throughput_mean_ops_sec",
            "throughput_ci95",
            "p50_mean_ms",
            "p95_mean_ms",
            "p99_mean_ms",
            "avg_mean_ms",
            "peak_rss_mb_mean",
            "cpu_util_pct_mean",
        ])
        for a in aggregates:
            writer.writerow([
                a.framework,
                a.scenario,
                a.samples,
                a.iterations_per_sample,
                f"{a.startup_ms_mean:.4f}",
                f"{a.startup_ms_p95:.4f}",
                f"{a.first_run_ms_mean:.4f}",
                f"{a.throughput_mean:.4f}",
                f"{a.throughput_ci95:.4f}",
                f"{a.p50_mean:.4f}",
                f"{a.p95_mean:.4f}",
                f"{a.p99_mean:.4f}",
                f"{a.avg_ms_mean:.4f}",
                f"{a.peak_rss_mb_mean:.4f}",
                f"{a.cpu_util_pct_mean:.2f}",
            ])

    print(f"wrote: {args.out_json}")
    print(f"wrote: {args.out_csv}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

# Development Blueprint (from Dev Team report)

Source: `../AgentRouter_DevTeam_Report.txt`

## Non-negotiables

- Determinism: identical input/context must produce identical execution path.
- Observability: every invocation must be traceable and measurable.
- Concurrency: router controls goroutine-based parallel execution.
- Transparency: no hidden framework state.

## Canonical contracts

```go
package agentfunc

import (
    "context"
    "time"
)

type AgentFunc func(ctx context.Context, input AgentInput) (AgentOutput, error)

type AgentInput struct {
    TaskID    string
    RequestID string
    Payload   []byte
    Metadata  map[string]string
    Timestamp time.Time
}

type AgentOutput struct {
    RequestID string
    Payload   []byte
    Metadata  map[string]string
    Duration  time.Duration
}

type RouterConfig struct {
    WorkerPoolSize int
    ChannelBuffer  int
    DefaultTimeout time.Duration
    RetryPolicy    RetryPolicy
}

type RetryPolicy struct {
    MaxAttempts   int
    Backoff       BackoffStrategy
    RetryableErrs []error
}
```

## Layered implementation model

1. Layer 1: Agent function interface + registry
2. Layer 2: Router engine (dispatcher/executor/aggregator)
3. Layer 3: Explicit immutable state via context
4. Layer 4: Observability (logs, traces, metrics, replay)

## Engineering constraints

- Router is the only owner of goroutine creation.
- Agents must be goroutine-safe and must respect `ctx.Done()`.
- Agents must not spawn goroutines internally.
- No global mutable state in internal packages.
- Channels are buffered by default.
- Manifests are validated at startup.

## Testing strategy

- `tests/unit`: contract-level behavior and helper functions
- `tests/integration`: end-to-end pipeline behavior
- `tests/replay`: deterministic replay output/order checks
- Benchmarks: throughput and P99 latency on core router paths

## Build/deploy baseline

- Go 1.22+
- Static Linux build (`CGO_ENABLED=0`)
- Scratch image target for containerized deployment


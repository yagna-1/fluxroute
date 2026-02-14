package metrics

import (
	"sync"
	"time"
)

// Recorder defines minimal metric hooks for router instrumentation.
type Recorder interface {
	ObserveInvocation(agentID string, status string, duration time.Duration)
	ObserveRetry(agentID string)
}

// NoopRecorder is default until Prometheus integration is added.
type NoopRecorder struct{}

func (NoopRecorder) ObserveInvocation(string, string, time.Duration) {}
func (NoopRecorder) ObserveRetry(string)                             {}

// Snapshot contains aggregated in-memory runtime metrics.
type Snapshot struct {
	TotalInvocations int
	ErrorInvocations int
	RetryAttempts    int
	ByAgent          map[string]AgentStats
}

// AgentStats is per-agent invocation telemetry.
type AgentStats struct {
	Successes     int
	Errors        int
	Retries       int
	TotalDuration time.Duration
}

// InMemoryRecorder records metrics in-process for local observability/testing.
type InMemoryRecorder struct {
	mu      sync.Mutex
	byAgent map[string]AgentStats
	total   int
	errors  int
	retries int
}

func NewInMemoryRecorder() *InMemoryRecorder {
	return &InMemoryRecorder{byAgent: make(map[string]AgentStats)}
}

func (r *InMemoryRecorder) ObserveInvocation(agentID string, status string, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.byAgent[agentID]
	if status == "success" {
		s.Successes++
	} else {
		s.Errors++
		r.errors++
	}
	s.TotalDuration += duration
	r.byAgent[agentID] = s
	r.total++
}

func (r *InMemoryRecorder) ObserveRetry(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.byAgent[agentID]
	s.Retries++
	r.byAgent[agentID] = s
	r.retries++
}

func (r *InMemoryRecorder) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	byAgent := make(map[string]AgentStats, len(r.byAgent))
	for k, v := range r.byAgent {
		byAgent[k] = v
	}
	return Snapshot{
		TotalInvocations: r.total,
		ErrorInvocations: r.errors,
		RetryAttempts:    r.retries,
		ByAgent:          byAgent,
	}
}

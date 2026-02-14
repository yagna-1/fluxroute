package metrics

import "time"

// Recorder defines minimal metric hooks for router instrumentation.
type Recorder interface {
	ObserveInvocation(agentID string, status string, duration time.Duration)
	ObserveRetry(agentID string)
}

// NoopRecorder is default until Prometheus integration is added.
type NoopRecorder struct{}

func (NoopRecorder) ObserveInvocation(string, string, time.Duration) {}
func (NoopRecorder) ObserveRetry(string)                             {}

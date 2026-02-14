package metrics

import "time"

// MultiRecorder fans out metrics to multiple recorders.
type MultiRecorder struct {
	recorders []Recorder
}

func NewMultiRecorder(recorders ...Recorder) *MultiRecorder {
	nonNil := make([]Recorder, 0, len(recorders))
	for _, r := range recorders {
		if r != nil {
			nonNil = append(nonNil, r)
		}
	}
	return &MultiRecorder{recorders: nonNil}
}

func (m *MultiRecorder) ObserveInvocation(agentID string, status string, duration time.Duration) {
	for _, r := range m.recorders {
		r.ObserveInvocation(agentID, status, duration)
	}
}

func (m *MultiRecorder) ObserveRetry(agentID string) {
	for _, r := range m.recorders {
		r.ObserveRetry(agentID)
	}
}

func (m *MultiRecorder) ObserveCircuitOpen(agentID string) {
	for _, r := range m.recorders {
		r.ObserveCircuitOpen(agentID)
	}
}

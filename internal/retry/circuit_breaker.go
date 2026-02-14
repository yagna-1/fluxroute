package retry

import (
	"sync"
	"time"

	"github.com/your-org/fluxroute/pkg/agentfunc"
)

// CircuitBreaker maintains per-agent breaker state.
type CircuitBreaker struct {
	mu     sync.Mutex
	states map[string]circuitState
}

type circuitState struct {
	consecutiveFailures int
	openUntil           time.Time
	halfOpenProbeActive bool
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{states: make(map[string]circuitState)}
}

// Allow decides whether a request should proceed.
// The second return value indicates whether the request is a half-open probe.
func (cb *CircuitBreaker) Allow(agentID string, policy agentfunc.CircuitBreakerPolicy, now time.Time) (bool, bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if policy.FailureThreshold <= 0 {
		return true, false
	}

	s := cb.states[agentID]
	if s.halfOpenProbeActive {
		return false, false
	}
	if s.openUntil.IsZero() {
		return true, false
	}
	if now.Before(s.openUntil) {
		return false, false
	}

	// Half-open transition: allow exactly one trial request.
	s.openUntil = time.Time{}
	s.consecutiveFailures = 0
	s.halfOpenProbeActive = true
	cb.states[agentID] = s
	return true, true
}

func (cb *CircuitBreaker) RecordSuccess(agentID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	s := cb.states[agentID]
	s.consecutiveFailures = 0
	s.openUntil = time.Time{}
	s.halfOpenProbeActive = false
	cb.states[agentID] = s
}

func (cb *CircuitBreaker) RecordFailure(agentID string, policy agentfunc.CircuitBreakerPolicy, now time.Time) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if policy.FailureThreshold <= 0 {
		return
	}
	if policy.ResetTimeout <= 0 {
		policy.ResetTimeout = 60 * time.Second
	}

	s := cb.states[agentID]
	if s.halfOpenProbeActive {
		s.openUntil = now.Add(policy.ResetTimeout)
		s.consecutiveFailures = 0
		s.halfOpenProbeActive = false
		cb.states[agentID] = s
		return
	}

	s.consecutiveFailures++
	if s.consecutiveFailures >= policy.FailureThreshold {
		s.openUntil = now.Add(policy.ResetTimeout)
		s.consecutiveFailures = 0
		s.halfOpenProbeActive = false
	}
	cb.states[agentID] = s
}

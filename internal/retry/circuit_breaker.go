package retry

import (
	"sync"
	"time"

	"github.com/your-org/agent-router/pkg/agentfunc"
)

// CircuitBreaker maintains per-agent breaker state.
type CircuitBreaker struct {
	mu     sync.Mutex
	states map[string]circuitState
}

type circuitState struct {
	consecutiveFailures int
	openUntil           time.Time
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{states: make(map[string]circuitState)}
}

func (cb *CircuitBreaker) Allow(agentID string, policy agentfunc.CircuitBreakerPolicy, now time.Time) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if policy.FailureThreshold <= 0 {
		return true
	}

	s := cb.states[agentID]
	if s.openUntil.IsZero() {
		return true
	}
	if now.Before(s.openUntil) {
		return false
	}

	// Half-open transition: allow a trial request and reset counters.
	s.openUntil = time.Time{}
	s.consecutiveFailures = 0
	cb.states[agentID] = s
	return true
}

func (cb *CircuitBreaker) RecordSuccess(agentID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	s := cb.states[agentID]
	s.consecutiveFailures = 0
	s.openUntil = time.Time{}
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
	s.consecutiveFailures++
	if s.consecutiveFailures >= policy.FailureThreshold {
		s.openUntil = now.Add(policy.ResetTimeout)
		s.consecutiveFailures = 0
	}
	cb.states[agentID] = s
}

package unit

import (
	"testing"
	"time"

	"github.com/your-org/fluxroute/internal/retry"
	"github.com/your-org/fluxroute/pkg/agentfunc"
)

func TestCircuitBreakerOpensAndResets(t *testing.T) {
	cb := retry.NewCircuitBreaker()
	policy := agentfunc.CircuitBreakerPolicy{FailureThreshold: 1, ResetTimeout: 50 * time.Millisecond}

	now := time.Now()
	if allow, _ := cb.Allow("agent_a", policy, now); !allow {
		t.Fatal("breaker should allow initial call")
	}

	cb.RecordFailure("agent_a", policy, now)
	if allow, _ := cb.Allow("agent_a", policy, now.Add(10*time.Millisecond)); allow {
		t.Fatal("breaker should be open after threshold reached")
	}

	allow, probe := cb.Allow("agent_a", policy, now.Add(60*time.Millisecond))
	if !allow || !probe {
		t.Fatal("breaker should reset after timeout")
	}
}

func TestCircuitBreakerHalfOpenAllowsSingleProbe(t *testing.T) {
	cb := retry.NewCircuitBreaker()
	policy := agentfunc.CircuitBreakerPolicy{FailureThreshold: 1, ResetTimeout: 10 * time.Millisecond}
	now := time.Now()

	cb.RecordFailure("agent_b", policy, now)

	allow, probe := cb.Allow("agent_b", policy, now.Add(15*time.Millisecond))
	if !allow || !probe {
		t.Fatal("expected first post-reset request to be a half-open probe")
	}

	allow, _ = cb.Allow("agent_b", policy, now.Add(16*time.Millisecond))
	if allow {
		t.Fatal("expected concurrent half-open probe to be blocked")
	}
}

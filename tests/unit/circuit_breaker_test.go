package unit

import (
	"testing"
	"time"

	"github.com/your-org/agent-router/internal/retry"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

func TestCircuitBreakerOpensAndResets(t *testing.T) {
	cb := retry.NewCircuitBreaker()
	policy := agentfunc.CircuitBreakerPolicy{FailureThreshold: 1, ResetTimeout: 50 * time.Millisecond}

	now := time.Now()
	if !cb.Allow("agent_a", policy, now) {
		t.Fatal("breaker should allow initial call")
	}

	cb.RecordFailure("agent_a", policy, now)
	if cb.Allow("agent_a", policy, now.Add(10*time.Millisecond)) {
		t.Fatal("breaker should be open after threshold reached")
	}

	if !cb.Allow("agent_a", policy, now.Add(60*time.Millisecond)) {
		t.Fatal("breaker should reset after timeout")
	}
}

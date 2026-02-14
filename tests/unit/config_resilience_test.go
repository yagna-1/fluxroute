package unit

import (
	"testing"
	"time"

	"github.com/your-org/fluxroute/internal/config"
	"github.com/your-org/fluxroute/pkg/agentfunc"
)

func TestCircuitBreakerPolicyFromConfigParsesProbeTimeout(t *testing.T) {
	fallback := agentfunc.CircuitBreakerPolicy{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		ProbeTimeout:     5 * time.Second,
	}
	cfg := config.CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     "45s",
		ProbeTimeout:     "7s",
	}

	got, err := config.CircuitBreakerPolicyFromConfig(cfg, fallback)
	if err != nil {
		t.Fatalf("parse circuit policy: %v", err)
	}
	if got.FailureThreshold != 5 {
		t.Fatalf("expected failure_threshold=5, got %d", got.FailureThreshold)
	}
	if got.ResetTimeout != 45*time.Second {
		t.Fatalf("expected reset_timeout=45s, got %s", got.ResetTimeout)
	}
	if got.ProbeTimeout != 7*time.Second {
		t.Fatalf("expected probe_timeout=7s, got %s", got.ProbeTimeout)
	}
}

func TestCircuitBreakerPolicyFromConfigDefaultsProbeTimeout(t *testing.T) {
	fallback := agentfunc.CircuitBreakerPolicy{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		ProbeTimeout:     0,
	}
	got, err := config.CircuitBreakerPolicyFromConfig(config.CircuitBreakerConfig{}, fallback)
	if err != nil {
		t.Fatalf("parse default policy: %v", err)
	}
	if got.ProbeTimeout <= 0 {
		t.Fatalf("expected default probe timeout to be set, got %s", got.ProbeTimeout)
	}
}

package unit

import (
	"testing"

	"github.com/your-org/agent-router/internal/config"
)

func TestValidateManifest(t *testing.T) {
	m := config.Manifest{
		Agents: []config.AgentBinding{{ID: "summarize_agent"}, {ID: "classify_agent"}},
		Pipeline: []config.PipelineStep{
			{Step: "summarize_agent"},
			{Step: "classify_agent", DependsOn: "summarize_agent"},
		},
	}

	if err := config.ValidateManifest(m); err != nil {
		t.Fatalf("expected valid manifest, got error: %v", err)
	}
}

func TestValidateManifestRejectsMissingAgent(t *testing.T) {
	m := config.Manifest{
		Agents: []config.AgentBinding{{ID: "summarize_agent"}},
		Pipeline: []config.PipelineStep{
			{Step: "summarize_agent"},
			{Step: "classify_agent", DependsOn: "summarize_agent"},
		},
	}

	if err := config.ValidateManifest(m); err == nil {
		t.Fatal("expected error for step without matching agent")
	}
}

func TestOrderedPipelineDetectsCycle(t *testing.T) {
	m := config.Manifest{
		Agents: []config.AgentBinding{{ID: "a"}, {ID: "b"}},
		Pipeline: []config.PipelineStep{
			{Step: "a", DependsOn: "b"},
			{Step: "b", DependsOn: "a"},
		},
	}

	if _, err := config.OrderedPipeline(m); err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestOrderedPipelineDeterministic(t *testing.T) {
	m := config.Manifest{
		Agents: []config.AgentBinding{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
		Pipeline: []config.PipelineStep{
			{Step: "a"},
			{Step: "b", DependsOn: "a"},
			{Step: "c"},
			{Step: "d", DependsOn: "c"},
		},
	}

	ordered, err := config.OrderedPipeline(m)
	if err != nil {
		t.Fatalf("unexpected ordering error: %v", err)
	}

	if len(ordered) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(ordered))
	}

	// Kahn queue is seeded in manifest order for deterministic tie-breaking.
	got := []string{ordered[0].Step, ordered[1].Step, ordered[2].Step, ordered[3].Step}
	want := []string{"a", "c", "b", "d"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected order at %d: got=%v want=%v", i, got, want)
		}
	}
}

func TestValidateManifestRejectsInvalidCircuitResetTimeout(t *testing.T) {
	m := config.Manifest{
		Agents: []config.AgentBinding{{
			ID: "a",
			CircuitBreaker: config.CircuitBreakerConfig{
				FailureThreshold: 1,
				ResetTimeout:     "not-a-duration",
			},
		}},
		Pipeline: []config.PipelineStep{
			{Step: "a"},
		},
	}

	if err := config.ValidateManifest(m); err == nil {
		t.Fatal("expected invalid circuit breaker duration error")
	}
}

func TestValidateManifestRejectsInvalidRBACRole(t *testing.T) {
	m := config.Manifest{
		Router: config.RouterSettings{
			RBAC: config.RBAC{
				RunRoles: []string{"not-a-role"},
			},
		},
		Agents: []config.AgentBinding{{ID: "a"}},
		Pipeline: []config.PipelineStep{
			{Step: "a"},
		},
	}

	if err := config.ValidateManifest(m); err == nil {
		t.Fatal("expected invalid rbac role error")
	}
}

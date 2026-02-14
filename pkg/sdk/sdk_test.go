package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/your-org/fluxroute/pkg/agentfunc"
)

func TestRuntimeRunPlan(t *testing.T) {
	r := NewRuntime(agentfunc.RouterConfig{DefaultTimeout: time.Second})
	if err := r.RegisterAgent("a", func(_ context.Context, in agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: in.RequestID, Payload: []byte("A")}, nil
	}); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := r.RegisterAgent("b", func(_ context.Context, in agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: in.RequestID, Payload: []byte("B")}, nil
	}); err != nil {
		t.Fatalf("register b: %v", err)
	}

	results, tr, err := r.RunPlan(context.Background(), "task_1", []Node{
		{ID: "001_a", AgentID: "a", Input: agentfunc.AgentInput{RequestID: "req_a"}},
		{ID: "002_b", AgentID: "b", Input: agentfunc.AgentInput{RequestID: "req_b"}, DependsOn: []string{"001_a"}},
	})
	if err != nil {
		t.Fatalf("run plan: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].InvocationID != "001_a" || results[1].InvocationID != "002_b" {
		t.Fatalf("unexpected order: %+v", results)
	}
	if tr.TaskID != "task_1" || len(tr.Steps) == 0 {
		t.Fatalf("unexpected trace: %+v", tr)
	}
}

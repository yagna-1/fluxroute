package unit

import (
	"context"
	"errors"
	"testing"

	"github.com/your-org/fluxroute/internal/agent"
	"github.com/your-org/fluxroute/pkg/agentfunc"
)

func TestRegistryRegisterAndGetDefaultVersion(t *testing.T) {
	reg := agent.NewRegistry()
	fn := func(_ context.Context, in agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: in.RequestID, Payload: []byte("ok")}, nil
	}

	if err := reg.Register("summarize_agent", fn); err != nil {
		t.Fatalf("register default version failed: %v", err)
	}

	got, ok := reg.Get("summarize_agent")
	if !ok {
		t.Fatal("expected registered agent")
	}
	if _, err := got(context.Background(), agentfunc.AgentInput{RequestID: "r1"}); err != nil {
		t.Fatalf("invoke default version failed: %v", err)
	}
}

func TestRegistryRegisterVersionAndGetVersion(t *testing.T) {
	reg := agent.NewRegistry()
	v1 := func(_ context.Context, in agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: in.RequestID, Payload: []byte("v1")}, nil
	}
	v2 := func(_ context.Context, in agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: in.RequestID, Payload: []byte("v2")}, nil
	}

	if err := reg.RegisterVersion("summarize_agent", "v1", v1); err != nil {
		t.Fatalf("register v1 failed: %v", err)
	}
	if err := reg.RegisterVersion("summarize_agent", "v2", v2); err != nil {
		t.Fatalf("register v2 failed: %v", err)
	}

	fn, ok := reg.GetVersion("summarize_agent", "v2")
	if !ok {
		t.Fatal("expected v2 registration")
	}
	out, err := fn(context.Background(), agentfunc.AgentInput{RequestID: "r2"})
	if err != nil {
		t.Fatalf("invoke v2 failed: %v", err)
	}
	if string(out.Payload) != "v2" {
		t.Fatalf("expected v2 payload, got %q", string(out.Payload))
	}
}

func TestRegistryVersionedDuplicateAndValidationErrors(t *testing.T) {
	reg := agent.NewRegistry()
	fn := func(_ context.Context, in agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: in.RequestID}, nil
	}

	if err := reg.RegisterVersion("agent_a", "v3", fn); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := reg.RegisterVersion("agent_a", "v3", fn); !errors.Is(err, agent.ErrDuplicateAgentID) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	if err := reg.RegisterVersion("", "v1", fn); !errors.Is(err, agent.ErrEmptyAgentID) {
		t.Fatalf("expected empty id error, got %v", err)
	}
	if err := reg.RegisterVersion("agent_b", "v1", nil); !errors.Is(err, agent.ErrNilAgentFunc) {
		t.Fatalf("expected nil fn error, got %v", err)
	}
}

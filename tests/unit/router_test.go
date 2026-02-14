package unit

import (
	"context"
	"testing"
	"time"

	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/internal/router"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

func TestEngineRunDeterministicOrder(t *testing.T) {
	reg := agent.NewRegistry()
	err := reg.Register("echo", func(_ context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: input.Payload}, nil
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	eng := router.NewEngine(reg, agentfunc.RouterConfig{ChannelBuffer: 2, DefaultTimeout: time.Second})
	results := eng.Run(context.Background(), []router.AgentInvocation{
		{ID: "b", AgentID: "echo", Input: agentfunc.AgentInput{RequestID: "2", Payload: []byte("two")}},
		{ID: "a", AgentID: "echo", Input: agentfunc.AgentInput{RequestID: "1", Payload: []byte("one")}},
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Invocation.ID != "a" || results[1].Invocation.ID != "b" {
		t.Fatalf("results not sorted deterministically: %+v", results)
	}
}

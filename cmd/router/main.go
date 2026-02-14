package main

import (
	"context"
	"fmt"
	"time"

	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/internal/config"
	"github.com/your-org/agent-router/internal/router"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

func main() {
	registry := agent.NewRegistry()
	_ = registry.Register("echo_agent", func(ctx context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		select {
		case <-ctx.Done():
			return agentfunc.AgentOutput{}, ctx.Err()
		default:
		}
		return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: input.Payload}, nil
	})

	engine := router.NewEngine(registry, config.FromEnv())
	results := engine.Run(context.Background(), []router.AgentInvocation{{
		ID:      "1",
		AgentID: "echo_agent",
		Input: agentfunc.AgentInput{
			TaskID:    "task_demo",
			RequestID: "req_1",
			Payload:   []byte(`{"message":"hello"}`),
			Timestamp: time.Now(),
		},
	}})

	fmt.Printf("router executed %d invocation(s)\n", len(results))
}

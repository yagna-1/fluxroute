package app

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/internal/config"
	"github.com/your-org/agent-router/internal/router"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

// RunManifest loads a manifest, executes the pipeline, and writes a summary.
func RunManifest(manifestPath string, out io.Writer) error {
	manifest, err := config.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	orderedSteps, err := config.OrderedPipeline(manifest)
	if err != nil {
		return fmt.Errorf("order pipeline: %w", err)
	}

	registry := agent.NewRegistry()
	for _, a := range manifest.Agents {
		agentID := a.ID
		err := registry.Register(agentID, func(ctx context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
			select {
			case <-ctx.Done():
				return agentfunc.AgentOutput{}, ctx.Err()
			default:
			}
			payload := []byte(fmt.Sprintf("{\"agent\":\"%s\",\"input\":%q}", agentID, string(input.Payload)))
			return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: payload}, nil
		})
		if err != nil {
			return fmt.Errorf("register agent %q: %w", agentID, err)
		}
	}

	engine := router.NewEngine(registry, config.FromEnv())
	taskID := "task_demo"
	invocations := make([]router.AgentInvocation, 0, len(orderedSteps))
	for i, step := range orderedSteps {
		id := fmt.Sprintf("%04d_%s", i+1, step.Step)
		invocations = append(invocations, router.AgentInvocation{
			ID:      id,
			AgentID: step.Step,
			Input: agentfunc.AgentInput{
				TaskID:    taskID,
				RequestID: fmt.Sprintf("req_%04d", i+1),
				Payload:   []byte(`{"message":"hello"}`),
				Metadata: map[string]string{
					"pipeline_step": step.Step,
				},
				Timestamp: time.Now(),
			},
		})
	}

	results := engine.Run(context.Background(), invocations)
	fmt.Fprintf(out, "router executed %d invocation(s) from %s\n", len(results), manifestPath)
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(out, "- %s (%s): error=%v\n", r.Invocation.ID, r.Invocation.AgentID, r.Err)
			continue
		}
		fmt.Fprintf(out, "- %s (%s): ok duration=%s\n", r.Invocation.ID, r.Invocation.AgentID, r.Output.Duration)
	}
	return nil
}

// ValidateManifest loads and validates a manifest only.
func ValidateManifest(manifestPath string) error {
	_, err := config.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("validate manifest: %w", err)
	}
	return nil
}

package sdk

import (
	"context"
	"fmt"

	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/internal/router"
	intracetrace "github.com/your-org/agent-router/internal/trace"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

// Node defines one planned invocation in the SDK surface.
type Node struct {
	ID                   string
	AgentID              string
	Input                agentfunc.AgentInput
	DependsOn            []string
	RetryPolicy          agentfunc.RetryPolicy
	CircuitBreakerPolicy agentfunc.CircuitBreakerPolicy
}

// Result is the SDK-friendly invocation result.
type Result struct {
	InvocationID string
	AgentID      string
	Output       agentfunc.AgentOutput
	Error        string
}

// Trace is the SDK-friendly trace surface.
type Trace struct {
	TaskID       string
	Steps        []TraceStep
	TotalLatency int64
}

// TraceStep is one replayable step in the SDK trace.
type TraceStep struct {
	InvocationID string
	AgentID      string
	RequestID    string
	Error        string
	Attempt      int
}

// Runtime provides public API access over the internal execution engine.
type Runtime struct {
	registry *agent.Registry
	engine   *router.Engine
}

// NewRuntime creates a runtime with isolated registry and engine config.
func NewRuntime(cfg agentfunc.RouterConfig) *Runtime {
	reg := agent.NewRegistry()
	eng := router.NewEngine(reg, cfg)
	return &Runtime{registry: reg, engine: eng}
}

// RegisterAgent registers an AgentFunc under the provided ID.
func (r *Runtime) RegisterAgent(agentID string, fn agentfunc.AgentFunc) error {
	return r.registry.Register(agentID, fn)
}

// RunPlan executes a dependency-aware plan and returns SDK-friendly results/trace.
func (r *Runtime) RunPlan(ctx context.Context, taskID string, nodes []Node) ([]Result, Trace, error) {
	if len(nodes) == 0 {
		return nil, Trace{}, fmt.Errorf("sdk: no nodes provided")
	}

	planNodes := make([]router.PlanNode, 0, len(nodes))
	for i, n := range nodes {
		id := n.ID
		if id == "" {
			id = fmt.Sprintf("%04d_%s", i+1, n.AgentID)
		}
		planNodes = append(planNodes, router.PlanNode{
			Invocation: router.AgentInvocation{
				ID:      id,
				AgentID: n.AgentID,
				Input:   n.Input,
			},
			DependsOn:            append([]string(nil), n.DependsOn...),
			RetryPolicy:          n.RetryPolicy,
			CircuitBreakerPolicy: n.CircuitBreakerPolicy,
		})
	}

	results, tr := r.engine.RunPlan(ctx, router.ExecutionPlan{TaskID: taskID, Nodes: planNodes})
	outResults := make([]Result, 0, len(results))
	for _, rr := range results {
		errText := ""
		if rr.Err != nil {
			errText = rr.Err.Error()
		}
		outResults = append(outResults, Result{
			InvocationID: rr.Invocation.ID,
			AgentID:      rr.Invocation.AgentID,
			Output:       rr.Output,
			Error:        errText,
		})
	}
	return outResults, toSDKTrace(tr), nil
}

func toSDKTrace(in intracetrace.ExecutionTrace) Trace {
	steps := make([]TraceStep, 0, len(in.Steps))
	for _, s := range in.Steps {
		steps = append(steps, TraceStep{
			InvocationID: s.InvocationID,
			AgentID:      s.AgentID,
			RequestID:    s.RequestID,
			Error:        s.Error,
			Attempt:      s.Attempt,
		})
	}
	return Trace{TaskID: in.TaskID, Steps: steps, TotalLatency: in.TotalLatency.Milliseconds()}
}

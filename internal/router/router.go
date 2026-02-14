package router

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

// AgentInvocation represents a single scheduled agent call.
type AgentInvocation struct {
	ID      string
	AgentID string
	Input   agentfunc.AgentInput
}

// AgentResult is the execution outcome for one invocation.
type AgentResult struct {
	Invocation AgentInvocation
	Output     agentfunc.AgentOutput
	Err        error
}

// Engine coordinates agent execution.
type Engine struct {
	registry *agent.Registry
	cfg      agentfunc.RouterConfig
}

func NewEngine(registry *agent.Registry, cfg agentfunc.RouterConfig) *Engine {
	if cfg.ChannelBuffer <= 0 {
		cfg.ChannelBuffer = 1
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	return &Engine{registry: registry, cfg: cfg}
}

// Run executes invocations concurrently and returns deterministic ordering by invocation ID.
func (e *Engine) Run(ctx context.Context, invocations []AgentInvocation) []AgentResult {
	resultCh := make(chan AgentResult, max(e.cfg.ChannelBuffer, len(invocations)))

	for _, inv := range invocations {
		inv := inv
		go func() {
			defer func() {
				if r := recover(); r != nil {
					resultCh <- AgentResult{Invocation: inv, Err: fmt.Errorf("agent panic: %v", r)}
				}
			}()

			fn, ok := e.registry.Get(inv.AgentID)
			if !ok {
				resultCh <- AgentResult{Invocation: inv, Err: fmt.Errorf("agent not registered: %s", inv.AgentID)}
				return
			}

			runCtx, cancel := context.WithTimeout(ctx, e.cfg.DefaultTimeout)
			defer cancel()

			start := time.Now()
			out, err := fn(runCtx, inv.Input)
			if err == nil {
				out.Duration = time.Since(start)
			}
			resultCh <- AgentResult{Invocation: inv, Output: out, Err: err}
		}()
	}

	results := make([]AgentResult, 0, len(invocations))
	for i := 0; i < len(invocations); i++ {
		results = append(results, <-resultCh)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Invocation.ID < results[j].Invocation.ID
	})
	return results
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

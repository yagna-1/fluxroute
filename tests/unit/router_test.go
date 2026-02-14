package unit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/internal/metrics"
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

func TestEngineRunPlanRetriesFlakyAgent(t *testing.T) {
	reg := agent.NewRegistry()
	attempts := 0
	err := reg.Register("flaky", func(_ context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		attempts++
		if attempts == 1 {
			return agentfunc.AgentOutput{}, errors.New("transient")
		}
		return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: []byte("ok")}, nil
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	eng := router.NewEngine(reg, agentfunc.RouterConfig{
		DefaultTimeout: time.Second,
		RetryPolicy: agentfunc.RetryPolicy{
			MaxAttempts: 2,
			Backoff:     agentfunc.BackoffLinear,
		},
	})

	results, tr := eng.RunPlan(context.Background(), router.ExecutionPlan{TaskID: "task_retry", Nodes: []router.PlanNode{{
		Invocation:  router.AgentInvocation{ID: "001_flaky", AgentID: "flaky", Input: agentfunc.AgentInput{RequestID: "req_1"}},
		RetryPolicy: agentfunc.RetryPolicy{MaxAttempts: 2, Backoff: agentfunc.BackoffLinear},
	}}})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Fatalf("expected retry success, got error: %v", results[0].Err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(tr.Steps) != 2 {
		t.Fatalf("expected 2 trace steps for retries, got %d", len(tr.Steps))
	}
}

func TestEngineRunPlanDependencyFailureSkipsChild(t *testing.T) {
	reg := agent.NewRegistry()
	_ = reg.Register("fail_a", func(context.Context, agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{}, errors.New("boom")
	})
	_ = reg.Register("child_b", func(context.Context, agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: "req_b", Payload: []byte("should-not-run")}, nil
	})
	_ = reg.Register("independent_c", func(_ context.Context, in agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: in.RequestID, Payload: []byte("ok")}, nil
	})

	eng := router.NewEngine(reg, agentfunc.RouterConfig{DefaultTimeout: time.Second})
	results, _ := eng.RunPlan(context.Background(), router.ExecutionPlan{TaskID: "task_deps", Nodes: []router.PlanNode{
		{Invocation: router.AgentInvocation{ID: "001_a", AgentID: "fail_a", Input: agentfunc.AgentInput{RequestID: "req_a"}}},
		{Invocation: router.AgentInvocation{ID: "002_b", AgentID: "child_b", Input: agentfunc.AgentInput{RequestID: "req_b"}}, DependsOn: []string{"001_a"}},
		{Invocation: router.AgentInvocation{ID: "003_c", AgentID: "independent_c", Input: agentfunc.AgentInput{RequestID: "req_c"}}},
	}})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Fatal("expected parent failure")
	}
	if results[1].Err == nil {
		t.Fatal("expected dependency failure on child")
	}
	if results[2].Err != nil {
		t.Fatalf("independent node should succeed, got %v", results[2].Err)
	}
}

func TestEngineRunPlanConvertsPanicToError(t *testing.T) {
	reg := agent.NewRegistry()
	_ = reg.Register("panic_agent", func(context.Context, agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		panic("kaboom")
	})

	eng := router.NewEngine(reg, agentfunc.RouterConfig{DefaultTimeout: time.Second})
	results, tr := eng.RunPlan(context.Background(), router.ExecutionPlan{TaskID: "task_panic", Nodes: []router.PlanNode{{
		Invocation: router.AgentInvocation{ID: "001_p", AgentID: "panic_agent", Input: agentfunc.AgentInput{RequestID: "req_p"}},
	}}})

	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("expected panic converted to error, results=%+v", results)
	}
	if len(tr.Steps) != 1 || tr.Steps[0].Error == "" {
		t.Fatalf("expected trace error for panic, trace=%+v", tr)
	}
}

func TestEngineCircuitBreakerOpensAfterFailures(t *testing.T) {
	reg := agent.NewRegistry()
	_ = reg.Register("fail_agent", func(context.Context, agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{}, errors.New("forced failure")
	})

	eng := router.NewEngine(reg, agentfunc.RouterConfig{
		DefaultTimeout: time.Second,
		RetryPolicy: agentfunc.RetryPolicy{
			MaxAttempts: 1,
			Backoff:     agentfunc.BackoffLinear,
		},
		CircuitBreaker: agentfunc.CircuitBreakerPolicy{
			FailureThreshold: 1,
			ResetTimeout:     time.Minute,
		},
	})
	memMetrics := metrics.NewInMemoryRecorder()
	eng.SetMetricsRecorder(memMetrics)

	firstResults, _ := eng.RunPlan(context.Background(), router.ExecutionPlan{TaskID: "task_cb_1", Nodes: []router.PlanNode{{
		Invocation: router.AgentInvocation{ID: "001", AgentID: "fail_agent", Input: agentfunc.AgentInput{RequestID: "req_1"}},
	}}})
	if len(firstResults) != 1 || firstResults[0].Err == nil {
		t.Fatalf("expected first invocation failure, got %+v", firstResults)
	}

	secondResults, tr := eng.RunPlan(context.Background(), router.ExecutionPlan{TaskID: "task_cb_2", Nodes: []router.PlanNode{{
		Invocation: router.AgentInvocation{ID: "001", AgentID: "fail_agent", Input: agentfunc.AgentInput{RequestID: "req_2"}},
	}}})
	if len(secondResults) != 1 || secondResults[0].Err == nil {
		t.Fatalf("expected second invocation error, got %+v", secondResults)
	}
	if !strings.Contains(secondResults[0].Err.Error(), "circuit open") {
		t.Fatalf("expected circuit open error, got %v", secondResults[0].Err)
	}
	if len(tr.Steps) == 0 || !strings.Contains(tr.Steps[0].Error, "circuit open") {
		t.Fatalf("expected circuit-open trace step, got %+v", tr.Steps)
	}

	snap := memMetrics.Snapshot()
	if snap.CircuitOpens != 1 {
		t.Fatalf("expected one circuit open metric, got %d", snap.CircuitOpens)
	}
}

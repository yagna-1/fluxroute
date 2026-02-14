package bench

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/your-org/fluxroute/internal/agent"
	"github.com/your-org/fluxroute/internal/router"
	"github.com/your-org/fluxroute/pkg/agentfunc"
)

func BenchmarkEngineRunPlan_Sequential10(b *testing.B) {
	eng := benchmarkEngine(8)
	plan := sequentialPlan(10)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.RunPlan(context.Background(), plan)
	}
}

func BenchmarkEngineRunPlan_Parallel100(b *testing.B) {
	eng := benchmarkEngine(32)
	plan := parallelPlan(100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.RunPlan(context.Background(), plan)
	}
}

func benchmarkEngine(workerPool int) *router.Engine {
	reg := agent.NewRegistry()
	_ = reg.Register("bench_agent", func(_ context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: input.Payload}, nil
	})

	cfg := agentfunc.RouterConfig{
		WorkerPoolSize: workerPool,
		ChannelBuffer:  workerPool * 2,
		DefaultTimeout: time.Second,
		RetryPolicy: agentfunc.RetryPolicy{
			MaxAttempts: 1,
			Backoff:     agentfunc.BackoffLinear,
		},
		CircuitBreaker: agentfunc.CircuitBreakerPolicy{
			FailureThreshold: 5,
			ResetTimeout:     time.Second,
		},
	}
	return router.NewEngine(reg, cfg)
}

func sequentialPlan(n int) router.ExecutionPlan {
	nodes := make([]router.PlanNode, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%04d", i+1)
		deps := []string{}
		if i > 0 {
			deps = append(deps, fmt.Sprintf("%04d", i))
		}
		nodes = append(nodes, router.PlanNode{
			Invocation: router.AgentInvocation{ID: id, AgentID: "bench_agent", Input: agentfunc.AgentInput{RequestID: id, Payload: []byte("x")}},
			DependsOn:  deps,
		})
	}
	return router.ExecutionPlan{TaskID: "bench_seq", Nodes: nodes}
}

func parallelPlan(n int) router.ExecutionPlan {
	nodes := make([]router.PlanNode, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%04d", i+1)
		nodes = append(nodes, router.PlanNode{
			Invocation: router.AgentInvocation{ID: id, AgentID: "bench_agent", Input: agentfunc.AgentInput{RequestID: id, Payload: []byte("x")}},
		})
	}
	return router.ExecutionPlan{TaskID: "bench_parallel", Nodes: nodes}
}

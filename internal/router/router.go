package router

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/internal/metrics"
	"github.com/your-org/agent-router/internal/retry"
	"github.com/your-org/agent-router/internal/trace"
	"github.com/your-org/agent-router/pkg/agentfunc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// AgentInvocation represents a single scheduled agent call.
type AgentInvocation struct {
	ID      string
	AgentID string
	Input   agentfunc.AgentInput
}

// PlanNode describes one invocation and its execution dependencies.
type PlanNode struct {
	Invocation           AgentInvocation
	DependsOn            []string
	RetryPolicy          agentfunc.RetryPolicy
	CircuitBreakerPolicy agentfunc.CircuitBreakerPolicy
}

// ExecutionPlan is the run-time DAG to execute.
type ExecutionPlan struct {
	TaskID string
	Nodes  []PlanNode
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
	metrics  metrics.Recorder
	breaker  *retry.CircuitBreaker
	tracer   oteltrace.Tracer
}

func NewEngine(registry *agent.Registry, cfg agentfunc.RouterConfig) *Engine {
	if cfg.ChannelBuffer <= 0 {
		cfg.ChannelBuffer = 1
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.WorkerPoolSize <= 0 {
		cfg.WorkerPoolSize = 1
	}
	if cfg.RetryPolicy.MaxAttempts <= 0 {
		cfg.RetryPolicy.MaxAttempts = 1
	}
	if cfg.RetryPolicy.Backoff == "" {
		cfg.RetryPolicy.Backoff = agentfunc.BackoffLinear
	}
	if cfg.CircuitBreaker.ResetTimeout <= 0 {
		cfg.CircuitBreaker.ResetTimeout = 60 * time.Second
	}
	if cfg.CircuitBreaker.FailureThreshold < 0 {
		cfg.CircuitBreaker.FailureThreshold = 0
	}
	return &Engine{
		registry: registry,
		cfg:      cfg,
		metrics:  metrics.NoopRecorder{},
		breaker:  retry.NewCircuitBreaker(),
		tracer:   otel.Tracer("agent-router"),
	}
}

func (e *Engine) SetMetricsRecorder(rec metrics.Recorder) {
	if rec == nil {
		e.metrics = metrics.NoopRecorder{}
		return
	}
	e.metrics = rec
}

func (e *Engine) SetTracer(t oteltrace.Tracer) {
	if t == nil {
		e.tracer = otel.Tracer("agent-router")
		return
	}
	e.tracer = t
}

// Run executes invocations concurrently and returns deterministic ordering by invocation ID.
func (e *Engine) Run(ctx context.Context, invocations []AgentInvocation) []AgentResult {
	nodes := make([]PlanNode, 0, len(invocations))
	for _, inv := range invocations {
		nodes = append(nodes, PlanNode{Invocation: inv})
	}
	results, _ := e.RunPlan(ctx, ExecutionPlan{TaskID: inferTaskID(invocations), Nodes: nodes})
	return results
}

// RunPlan executes a dependency-aware plan with retries and full execution trace.
func (e *Engine) RunPlan(ctx context.Context, plan ExecutionPlan) ([]AgentResult, trace.ExecutionTrace) {
	start := time.Now()
	recorder := trace.NewRecorder(plan.TaskID, start)

	graph, err := buildGraph(plan)
	if err != nil {
		recorder.AddStep(trace.Step{
			InvocationID: "plan_validation",
			AgentID:      "router",
			RequestID:    "",
			Error:        err.Error(),
			Attempt:      0,
		})
		return []AgentResult{{Err: err}}, recorder.Finalize(time.Now())
	}

	resultsByID := make(map[string]AgentResult, len(graph.nodes))
	for _, level := range graph.levels {
		levelResults := e.executeLevel(ctx, level, graph, resultsByID, recorder)
		for _, r := range levelResults {
			resultsByID[r.Invocation.ID] = r
		}
	}

	results := make([]AgentResult, 0, len(graph.nodes))
	for _, node := range graph.nodes {
		results = append(results, resultsByID[node.Invocation.ID])
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Invocation.ID < results[j].Invocation.ID
	})

	return results, recorder.Finalize(time.Now())
}

func (e *Engine) executeLevel(
	ctx context.Context,
	level []string,
	graph planGraph,
	resultsByID map[string]AgentResult,
	recorder *trace.Recorder,
) []AgentResult {
	resultCh := make(chan AgentResult, len(level))
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.cfg.WorkerPoolSize)

	for _, nodeID := range level {
		node := graph.nodesByID[nodeID]
		if depErr := dependencyError(node, graph, resultsByID); depErr != nil {
			r := AgentResult{Invocation: node.Invocation, Err: depErr}
			recorder.AddStep(trace.Step{
				InvocationID: node.Invocation.ID,
				AgentID:      node.Invocation.AgentID,
				RequestID:    node.Invocation.Input.RequestID,
				Input:        node.Invocation.Input,
				Error:        depErr.Error(),
				Attempt:      0,
			})
			resultCh <- r
			continue
		}

		wg.Add(1)
		go func(n PlanNode) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resultCh <- e.executeNode(ctx, n, recorder)
		}(node)
	}

	wg.Wait()
	close(resultCh)

	levelResults := make([]AgentResult, 0, len(level))
	for r := range resultCh {
		levelResults = append(levelResults, r)
	}
	sort.Slice(levelResults, func(i, j int) bool {
		return levelResults[i].Invocation.ID < levelResults[j].Invocation.ID
	})
	return levelResults
}

func (e *Engine) executeNode(ctx context.Context, node PlanNode, recorder *trace.Recorder) AgentResult {
	policy := node.RetryPolicy
	if policy.MaxAttempts <= 0 {
		policy = e.cfg.RetryPolicy
	}
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}
	if policy.Backoff == "" {
		policy.Backoff = agentfunc.BackoffLinear
	}

	cbPolicy := node.CircuitBreakerPolicy
	if cbPolicy.FailureThreshold <= 0 {
		cbPolicy = e.cfg.CircuitBreaker
	}
	if cbPolicy.ResetTimeout <= 0 {
		cbPolicy.ResetTimeout = 60 * time.Second
	}

	fn, ok := e.registry.Get(node.Invocation.AgentID)
	if !ok {
		err := fmt.Errorf("agent not registered: %s", node.Invocation.AgentID)
		e.metrics.ObserveInvocation(node.Invocation.AgentID, "error", 0)
		recorder.AddStep(trace.Step{
			InvocationID: node.Invocation.ID,
			AgentID:      node.Invocation.AgentID,
			RequestID:    node.Invocation.Input.RequestID,
			Input:        node.Invocation.Input,
			Error:        err.Error(),
			Attempt:      1,
		})
		return AgentResult{Invocation: node.Invocation, Err: err}
	}

	if !e.breaker.Allow(node.Invocation.AgentID, cbPolicy, time.Now()) {
		err := fmt.Errorf("circuit open: %s", node.Invocation.AgentID)
		e.metrics.ObserveInvocation(node.Invocation.AgentID, "circuit_open", 0)
		e.metrics.ObserveCircuitOpen(node.Invocation.AgentID)
		recorder.AddStep(trace.Step{
			InvocationID: node.Invocation.ID,
			AgentID:      node.Invocation.AgentID,
			RequestID:    node.Invocation.Input.RequestID,
			Input:        node.Invocation.Input,
			Error:        err.Error(),
			Attempt:      0,
		})
		return AgentResult{Invocation: node.Invocation, Err: err}
	}

	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		runCtx, cancel := context.WithTimeout(ctx, e.cfg.DefaultTimeout)
		runCtx, span := e.tracer.Start(runCtx, "agent.invoke",
			oteltrace.WithAttributes(
				attribute.String("task.id", node.Invocation.Input.TaskID),
				attribute.String("request.id", node.Invocation.Input.RequestID),
				attribute.String("invocation.id", node.Invocation.ID),
				attribute.String("agent.id", node.Invocation.AgentID),
				attribute.Int("agent.attempt", attempt),
			),
		)
		started := time.Now()
		out, err := safeCall(fn, runCtx, node.Invocation.Input)
		cancel()
		duration := time.Since(started)

		if err == nil {
			if out.Duration == 0 {
				out.Duration = duration
			}
			e.breaker.RecordSuccess(node.Invocation.AgentID)
			e.metrics.ObserveInvocation(node.Invocation.AgentID, "success", out.Duration)
			recorder.AddStep(trace.Step{
				InvocationID: node.Invocation.ID,
				AgentID:      node.Invocation.AgentID,
				RequestID:    node.Invocation.Input.RequestID,
				Input:        node.Invocation.Input,
				Output:       out,
				Duration:     duration,
				Attempt:      attempt,
			})
			span.SetAttributes(attribute.String("status", "success"))
			span.End()
			return AgentResult{Invocation: node.Invocation, Output: out}
		}

		lastErr = err
		e.breaker.RecordFailure(node.Invocation.AgentID, cbPolicy, time.Now())
		e.metrics.ObserveInvocation(node.Invocation.AgentID, "error", duration)
		recorder.AddStep(trace.Step{
			InvocationID: node.Invocation.ID,
			AgentID:      node.Invocation.AgentID,
			RequestID:    node.Invocation.Input.RequestID,
			Input:        node.Invocation.Input,
			Output:       out,
			Error:        err.Error(),
			Duration:     duration,
			Attempt:      attempt,
		})
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("status", "error"))
		span.End()

		if attempt == policy.MaxAttempts {
			break
		}
		e.metrics.ObserveRetry(node.Invocation.AgentID)
		select {
		case <-ctx.Done():
			return AgentResult{Invocation: node.Invocation, Err: ctx.Err()}
		case <-time.After(retry.BackoffDuration(policy.Backoff, attempt)):
		}
	}

	return AgentResult{Invocation: node.Invocation, Err: lastErr}
}

func safeCall(fn agentfunc.AgentFunc, ctx context.Context, in agentfunc.AgentInput) (out agentfunc.AgentOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("agent panic: %v", r)
		}
	}()
	return fn(ctx, in)
}

func dependencyError(node PlanNode, graph planGraph, results map[string]AgentResult) error {
	for _, depID := range node.DependsOn {
		depResult, ok := results[depID]
		if !ok {
			return fmt.Errorf("dependency result missing: %s", depID)
		}
		if depResult.Err != nil {
			return fmt.Errorf("dependency failed: %s: %v", depID, depResult.Err)
		}
		if _, exists := graph.nodesByID[depID]; !exists {
			return fmt.Errorf("dependency missing in graph: %s", depID)
		}
	}
	return nil
}

type planGraph struct {
	nodes     []PlanNode
	nodesByID map[string]PlanNode
	levels    [][]string
}

func buildGraph(plan ExecutionPlan) (planGraph, error) {
	if len(plan.Nodes) == 0 {
		return planGraph{}, errors.New("execution plan has no nodes")
	}

	nodesByID := make(map[string]PlanNode, len(plan.Nodes))
	inDegree := make(map[string]int, len(plan.Nodes))
	children := make(map[string][]string, len(plan.Nodes))
	for _, n := range plan.Nodes {
		if n.Invocation.ID == "" {
			return planGraph{}, errors.New("execution plan has node with empty invocation id")
		}
		if _, exists := nodesByID[n.Invocation.ID]; exists {
			return planGraph{}, fmt.Errorf("execution plan has duplicate invocation id %q", n.Invocation.ID)
		}
		nodesByID[n.Invocation.ID] = n
		inDegree[n.Invocation.ID] = len(n.DependsOn)
	}

	for _, n := range plan.Nodes {
		for _, depID := range n.DependsOn {
			if _, ok := nodesByID[depID]; !ok {
				return planGraph{}, fmt.Errorf("execution plan node %q depends on unknown invocation %q", n.Invocation.ID, depID)
			}
			if depID == n.Invocation.ID {
				return planGraph{}, fmt.Errorf("execution plan node %q depends on itself", n.Invocation.ID)
			}
			children[depID] = append(children[depID], n.Invocation.ID)
		}
	}

	queue := make([]string, 0)
	for _, n := range plan.Nodes {
		if inDegree[n.Invocation.ID] == 0 {
			queue = append(queue, n.Invocation.ID)
		}
	}
	sort.Strings(queue)

	visited := 0
	levels := make([][]string, 0)
	for len(queue) > 0 {
		level := append([]string(nil), queue...)
		levels = append(levels, level)
		visited += len(level)

		next := make([]string, 0)
		for _, curr := range level {
			for _, child := range children[curr] {
				inDegree[child]--
				if inDegree[child] == 0 {
					next = append(next, child)
				}
			}
		}
		sort.Strings(next)
		queue = next
	}

	if visited != len(plan.Nodes) {
		return planGraph{}, errors.New("execution plan contains cycle")
	}

	return planGraph{nodes: plan.Nodes, nodesByID: nodesByID, levels: levels}, nil
}

func inferTaskID(invocations []AgentInvocation) string {
	if len(invocations) == 0 {
		return ""
	}
	if invocations[0].Input.TaskID != "" {
		return invocations[0].Input.TaskID
	}
	return "task_default"
}

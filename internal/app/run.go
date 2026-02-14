package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/your-org/agent-router/internal/agent"
	"github.com/your-org/agent-router/internal/config"
	"github.com/your-org/agent-router/internal/metrics"
	"github.com/your-org/agent-router/internal/router"
	"github.com/your-org/agent-router/internal/trace"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

// RunReport captures the outputs from one manifest execution.
type RunReport struct {
	Results []router.AgentResult
	Trace   trace.ExecutionTrace
	Metrics metrics.Snapshot
}

// RunManifest loads a manifest, executes the pipeline, and writes a summary.
func RunManifest(manifestPath string, out io.Writer) error {
	report, err := RunManifestReport(manifestPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "router executed %d invocation(s) from %s\n", len(report.Results), manifestPath)
	failed := 0
	for _, r := range report.Results {
		if r.Err != nil {
			failed++
			fmt.Fprintf(out, "- %s (%s): error=%v\n", r.Invocation.ID, r.Invocation.AgentID, r.Err)
			continue
		}
		fmt.Fprintf(out, "- %s (%s): ok duration=%s\n", r.Invocation.ID, r.Invocation.AgentID, r.Output.Duration)
	}
	emitStructuredLogs(out, report)
	fmt.Fprintf(out, "metrics total_invocations=%d errors=%d retries=%d\n",
		report.Metrics.TotalInvocations,
		report.Metrics.ErrorInvocations,
		report.Metrics.RetryAttempts,
	)
	if report.Metrics.CircuitOpens > 0 {
		fmt.Fprintf(out, "metrics circuit_opens=%d\n", report.Metrics.CircuitOpens)
	}
	if failed > 0 {
		return fmt.Errorf("pipeline completed with %d failed invocation(s)", failed)
	}
	return nil
}

// RunManifestReport executes the manifest and returns results + trace.
func RunManifestReport(manifestPath string) (RunReport, error) {
	manifest, err := config.LoadManifest(manifestPath)
	if err != nil {
		return RunReport{}, fmt.Errorf("load manifest: %w", err)
	}

	registry, err := buildRegistry(manifest)
	if err != nil {
		return RunReport{}, err
	}

	baseCfg := config.FromEnv()
	runtimeCfg, err := config.RouterConfigFromManifest(manifest, baseCfg)
	if err != nil {
		return RunReport{}, fmt.Errorf("build runtime config: %w", err)
	}

	plan, err := buildExecutionPlan(manifest, runtimeCfg.CircuitBreaker)
	if err != nil {
		return RunReport{}, err
	}

	engine := router.NewEngine(registry, runtimeCfg)
	otelRuntime, err := trace.SetupOTelFromEnv("agent-router")
	if err != nil {
		return RunReport{}, fmt.Errorf("setup tracing: %w", err)
	}
	defer func() { _ = otelRuntime.Shutdown(context.Background()) }()
	engine.SetTracer(otelRuntime.Tracer)

	metricRecorder := metrics.NewInMemoryRecorder()
	activeRecorder := metrics.Recorder(metricRecorder)
	var metricsServer *http.Server
	if envBool("METRICS_ENABLED") {
		registry := prometheus.NewRegistry()
		promRecorder, err := metrics.NewPrometheusRecorder(registry)
		if err != nil {
			return RunReport{}, fmt.Errorf("setup prometheus recorder: %w", err)
		}
		activeRecorder = metrics.NewMultiRecorder(metricRecorder, promRecorder)
		metricsServer, err = metrics.StartPrometheusServer(metricsAddr(), registry)
		if err != nil {
			return RunReport{}, fmt.Errorf("start metrics endpoint: %w", err)
		}
		defer func() { _ = metrics.StopServer(context.Background(), metricsServer) }()
	}
	engine.SetMetricsRecorder(activeRecorder)
	results, execTrace := engine.RunPlan(context.Background(), plan)

	if tracePath := os.Getenv("TRACE_OUTPUT"); tracePath != "" {
		if err := trace.SaveToFile(tracePath, execTrace); err != nil {
			return RunReport{}, fmt.Errorf("persist trace: %w", err)
		}
	}

	return RunReport{Results: results, Trace: execTrace, Metrics: metricRecorder.Snapshot()}, nil
}

// ValidateManifest loads and validates a manifest only.
func ValidateManifest(manifestPath string) error {
	_, err := config.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("validate manifest: %w", err)
	}
	return nil
}

// ReplayTrace loads a trace and compares replay output against recorded output.
func ReplayTrace(tracePath string, out io.Writer) error {
	tr, err := trace.LoadFromFile(tracePath)
	if err != nil {
		return fmt.Errorf("load trace: %w", err)
	}

	registry := newGenericRegistry(uniqueAgentIDs(tr))
	resolver := func(agentID string) (agentfunc.AgentFunc, bool) {
		return registry.Get(agentID)
	}

	if err := trace.ReplayAndCompare(context.Background(), tr, 30*time.Second, resolver); err != nil {
		return fmt.Errorf("replay compare failed: %w", err)
	}
	fmt.Fprintf(out, "replay matched recorded outputs for %d step(s)\n", len(tr.Steps))
	return nil
}

func buildRegistry(manifest config.Manifest) (*agent.Registry, error) {
	registry := newGenericRegistry(nil)
	for _, a := range manifest.Agents {
		agentID := a.ID
		if err := registry.Register(agentID, deterministicAgent(agentID)); err != nil {
			return nil, fmt.Errorf("register agent %q: %w", agentID, err)
		}
	}
	return registry, nil
}

func newGenericRegistry(agentIDs []string) *agent.Registry {
	registry := agent.NewRegistry()
	for _, agentID := range agentIDs {
		_ = registry.Register(agentID, deterministicAgent(agentID))
	}
	return registry
}

func deterministicAgent(agentID string) agentfunc.AgentFunc {
	var mu sync.Mutex
	attemptsByRequest := map[string]int{}

	return func(ctx context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		select {
		case <-ctx.Done():
			return agentfunc.AgentOutput{}, ctx.Err()
		default:
		}

		mu.Lock()
		attemptsByRequest[input.RequestID]++
		attempt := attemptsByRequest[input.RequestID]
		mu.Unlock()

		switch {
		case strings.HasPrefix(agentID, "panic_"):
			panic("forced panic for test/runtime validation")
		case strings.HasPrefix(agentID, "fail_"):
			return agentfunc.AgentOutput{}, errors.New("forced failure")
		case strings.HasPrefix(agentID, "flaky_") && attempt == 1:
			return agentfunc.AgentOutput{}, errors.New("forced transient failure")
		}

		payload := []byte(fmt.Sprintf(
			"{\"agent\":\"%s\",\"input\":%q,\"attempt\":%d}",
			agentID,
			string(input.Payload),
			attempt,
		))
		return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: payload}, nil
	}
}

func buildExecutionPlan(manifest config.Manifest, defaultCB agentfunc.CircuitBreakerPolicy) (router.ExecutionPlan, error) {
	orderedSteps, err := config.OrderedPipeline(manifest)
	if err != nil {
		return router.ExecutionPlan{}, fmt.Errorf("order pipeline: %w", err)
	}

	retryByAgent := make(map[string]agentfunc.RetryPolicy, len(manifest.Agents))
	cbByAgent := make(map[string]agentfunc.CircuitBreakerPolicy, len(manifest.Agents))
	for _, a := range manifest.Agents {
		retryByAgent[a.ID] = config.RetryPolicyFromConfig(a.Retry)
		cbPolicy, err := config.CircuitBreakerPolicyFromConfig(a.CircuitBreaker, defaultCB)
		if err != nil {
			return router.ExecutionPlan{}, fmt.Errorf("agent %q circuit breaker policy: %w", a.ID, err)
		}
		cbByAgent[a.ID] = cbPolicy
	}

	invocationIDByStep := make(map[string]string, len(orderedSteps))
	nodes := make([]router.PlanNode, 0, len(orderedSteps))
	for i, step := range orderedSteps {
		invID := fmt.Sprintf("%04d_%s", i+1, step.Step)
		invocationIDByStep[step.Step] = invID
	}

	taskID := "task_demo"
	for i, step := range orderedSteps {
		depends := make([]string, 0, 1)
		if step.DependsOn != "" {
			depends = append(depends, invocationIDByStep[step.DependsOn])
		}
		nodes = append(nodes, router.PlanNode{
			Invocation: router.AgentInvocation{
				ID:      invocationIDByStep[step.Step],
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
			},
			DependsOn:            depends,
			RetryPolicy:          retryByAgent[step.Step],
			CircuitBreakerPolicy: cbByAgent[step.Step],
		})
	}

	return router.ExecutionPlan{TaskID: taskID, Nodes: nodes}, nil
}

func uniqueAgentIDs(tr trace.ExecutionTrace) []string {
	set := make(map[string]struct{})
	for _, s := range tr.Steps {
		if s.AgentID == "" || s.AgentID == "router" {
			continue
		}
		set[s.AgentID] = struct{}{}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func emitStructuredLogs(out io.Writer, report RunReport) {
	for _, r := range report.Results {
		status := "success"
		errText := ""
		if r.Err != nil {
			status = "error"
			errText = r.Err.Error()
		}

		entry := map[string]any{
			"level":       "info",
			"ts":          time.Now().UTC().Format(time.RFC3339Nano),
			"task_id":     r.Invocation.Input.TaskID,
			"request_id":  r.Invocation.Input.RequestID,
			"agent_id":    r.Invocation.AgentID,
			"attempt":     1,
			"duration_ms": r.Output.Duration.Milliseconds(),
			"status":      status,
		}
		if errText != "" {
			entry["error"] = errText
		}
		if b, err := json.Marshal(entry); err == nil {
			fmt.Fprintln(out, string(b))
		}
	}
}

func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func metricsAddr() string {
	if v := strings.TrimSpace(os.Getenv("METRICS_ADDR")); v != "" {
		return v
	}
	return ":2112"
}

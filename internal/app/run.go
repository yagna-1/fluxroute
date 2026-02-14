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
	"github.com/your-org/agent-router/internal/audit"
	"github.com/your-org/agent-router/internal/config"
	"github.com/your-org/agent-router/internal/coordinator"
	"github.com/your-org/agent-router/internal/metrics"
	"github.com/your-org/agent-router/internal/router"
	"github.com/your-org/agent-router/internal/security"
	"github.com/your-org/agent-router/internal/trace"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

// RunReport captures the outputs from one manifest execution.
type RunReport struct {
	Results   []router.AgentResult
	Trace     trace.ExecutionTrace
	Metrics   metrics.Snapshot
	Namespace string
}

// RunManifest loads a manifest, executes the pipeline, and writes a summary.
func RunManifest(manifestPath string, out io.Writer) error {
	report, err := RunManifestReport(manifestPath)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "router executed %d invocation(s) from %s (namespace=%s)\n", len(report.Results), manifestPath, report.Namespace)
	failed := 0
	for _, r := range report.Results {
		if r.Err != nil {
			failed++
			_, _ = fmt.Fprintf(out, "- %s (%s): error=%v\n", r.Invocation.ID, r.Invocation.AgentID, r.Err)
			continue
		}
		_, _ = fmt.Fprintf(out, "- %s (%s): ok duration=%s\n", r.Invocation.ID, r.Invocation.AgentID, r.Output.Duration)
	}
	emitStructuredLogs(out, report)
	_, _ = fmt.Fprintf(out, "metrics total_invocations=%d errors=%d retries=%d\n",
		report.Metrics.TotalInvocations,
		report.Metrics.ErrorInvocations,
		report.Metrics.RetryAttempts,
	)
	if report.Metrics.CircuitOpens > 0 {
		_, _ = fmt.Fprintf(out, "metrics circuit_opens=%d\n", report.Metrics.CircuitOpens)
	}
	if failed > 0 {
		return fmt.Errorf("pipeline completed with %d failed invocation(s)", failed)
	}
	return nil
}

// RunManifestReport executes the manifest and returns results + trace.
func RunManifestReport(manifestPath string) (report RunReport, retErr error) {
	logger := audit.NewLogger(strings.TrimSpace(os.Getenv("AUDIT_LOG_PATH")))
	actor := currentRole().String()
	defer func() {
		status := "success"
		if retErr != nil {
			status = "error"
		}
		_ = logger.Write(actor, string(security.ActionRun), manifestPath, status, retErr)
	}()

	manifest, err := config.LoadManifest(manifestPath)
	if err != nil {
		return RunReport{}, fmt.Errorf("load manifest: %w", err)
	}

	policy, err := config.RBACPolicyFromManifest(manifest)
	if err != nil {
		return RunReport{}, fmt.Errorf("build rbac policy: %w", err)
	}
	if err := authorize(policy, security.ActionRun); err != nil {
		return RunReport{}, err
	}

	namespace, err := config.NamespaceFromManifest(manifest)
	if err != nil {
		return RunReport{}, fmt.Errorf("namespace: %w", err)
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

	plan, err := buildExecutionPlan(manifest, namespace, runtimeCfg.CircuitBreaker)
	if err != nil {
		return RunReport{}, err
	}

	lease, err := acquireLeaseIfEnabled(context.Background(), namespace, plan.TaskID)
	if err != nil {
		return RunReport{}, err
	}
	if lease != nil {
		defer func() { _ = lease.Release(context.Background()) }()
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
		promRegistry := prometheus.NewRegistry()
		promRecorder, err := metrics.NewPrometheusRecorder(promRegistry)
		if err != nil {
			return RunReport{}, fmt.Errorf("setup prometheus recorder: %w", err)
		}
		activeRecorder = metrics.NewMultiRecorder(metricRecorder, promRecorder)
		if envBool("METRICS_TLS_ENABLED") {
			metricsServer, err = metrics.StartPrometheusServerTLS(
				metricsAddr(),
				promRegistry,
				os.Getenv("METRICS_TLS_CERT_FILE"),
				os.Getenv("METRICS_TLS_KEY_FILE"),
				os.Getenv("METRICS_TLS_CA_FILE"),
				envBool("METRICS_TLS_REQUIRE_CLIENT_CERT"),
			)
		} else {
			metricsServer, err = metrics.StartPrometheusServer(metricsAddr(), promRegistry)
		}
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

	return RunReport{Results: results, Trace: execTrace, Metrics: metricRecorder.Snapshot(), Namespace: namespace}, nil
}

// ValidateManifest loads and validates a manifest only.
func ValidateManifest(manifestPath string) (retErr error) {
	logger := audit.NewLogger(strings.TrimSpace(os.Getenv("AUDIT_LOG_PATH")))
	actor := currentRole().String()
	defer func() {
		status := "success"
		if retErr != nil {
			status = "error"
		}
		_ = logger.Write(actor, string(security.ActionValidate), manifestPath, status, retErr)
	}()

	manifest, err := config.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("validate manifest: %w", err)
	}
	policy, err := config.RBACPolicyFromManifest(manifest)
	if err != nil {
		return fmt.Errorf("validate manifest policy: %w", err)
	}
	if err := authorize(policy, security.ActionValidate); err != nil {
		return err
	}
	return nil
}

// ReplayTrace loads a trace and compares replay output against recorded output.
func ReplayTrace(tracePath string, out io.Writer) (retErr error) {
	logger := audit.NewLogger(strings.TrimSpace(os.Getenv("AUDIT_LOG_PATH")))
	actor := currentRole().String()
	defer func() {
		status := "success"
		if retErr != nil {
			status = "error"
		}
		_ = logger.Write(actor, string(security.ActionReplay), tracePath, status, retErr)
	}()

	if err := authorize(security.DefaultPolicy(), security.ActionReplay); err != nil {
		return err
	}

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
	_, _ = fmt.Fprintf(out, "replay matched recorded outputs for %d step(s)\n", len(tr.Steps))
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
		case strings.HasPrefix(agentID, "slow_"):
			select {
			case <-ctx.Done():
				return agentfunc.AgentOutput{}, ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
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

func buildExecutionPlan(manifest config.Manifest, namespace string, defaultCB agentfunc.CircuitBreakerPolicy) (router.ExecutionPlan, error) {
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

	taskID := namespace + ".task_demo"
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
						"namespace":     namespace,
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
			"namespace":   report.Namespace,
			"attempt":     1,
			"duration_ms": r.Output.Duration.Milliseconds(),
			"status":      status,
		}
		if errText != "" {
			entry["error"] = errText
		}
		if b, err := json.Marshal(entry); err == nil {
			_, _ = fmt.Fprintln(out, string(b))
		}
	}
}

func currentRole() security.Role {
	if r, err := security.ParseRole(os.Getenv("REQUEST_ROLE")); err == nil {
		return r
	}
	return security.RoleOperator
}

func authorize(policy security.Policy, action security.Action) error {
	role := currentRole()
	if !policy.IsAllowed(role, action) {
		return fmt.Errorf("rbac denied: role %q cannot perform %q", role, action)
	}
	return nil
}

func acquireLeaseIfEnabled(ctx context.Context, namespace string, taskID string) (coordinator.Lease, error) {
	if !envBool("COORDINATION_ENABLED") {
		return nil, nil
	}
	mode := strings.TrimSpace(strings.ToLower(os.Getenv("COORDINATION_MODE")))
	if mode == "" {
		mode = "file"
	}
	var coord coordinator.Coordinator
	switch mode {
	case "memory":
		coord = coordinator.NewMemoryCoordinator()
	case "redis":
		redisURL := strings.TrimSpace(os.Getenv("COORDINATION_REDIS_URL"))
		redisPrefix := strings.TrimSpace(os.Getenv("COORDINATION_REDIS_PREFIX"))
		redisCoord, err := coordinator.NewRedisCoordinator(redisURL, redisPrefix)
		if err != nil {
			return nil, err
		}
		coord = redisCoord
	default:
		coord = coordinator.NewFileCoordinator(os.Getenv("COORDINATION_DIR"))
	}

	ttl := 2 * time.Minute
	if v := strings.TrimSpace(os.Getenv("COORDINATION_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			ttl = d
		}
	}

	key := namespace + "-" + strings.ReplaceAll(taskID, ".", "_")
	lease, err := coord.Acquire(ctx, key, ttl)
	if err != nil {
		return nil, fmt.Errorf("coordination acquire failed: %w", err)
	}
	return lease, nil
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

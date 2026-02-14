package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Generate creates a minimal runnable project scaffold in targetDir.
func Generate(targetDir string, pipelineName string) error {
	if strings.TrimSpace(targetDir) == "" {
		return fmt.Errorf("target directory is empty")
	}
	if strings.TrimSpace(pipelineName) == "" {
		pipelineName = "sample"
	}

	agentA := pipelineName + "_step_a"
	agentB := pipelineName + "_step_b"

	if err := os.MkdirAll(filepath.Join(targetDir, "agents"), 0o755); err != nil {
		return fmt.Errorf("mkdir agents: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "manifests"), 0o755); err != nil {
		return fmt.Errorf("mkdir manifests: %w", err)
	}

	manifest := fmt.Sprintf(`router:
  worker_pool_size: 4
  channel_buffer: 16
  default_timeout: 10s
  namespace: %s
  rbac:
    run_roles: [operator, admin]
    validate_roles: [viewer, operator, admin]
    replay_roles: [operator, admin]

agents:
  - id: %s
    retry:
      max_attempts: 1
      backoff: linear
    circuit_breaker:
      failure_threshold: 3
      reset_timeout: 30s
  - id: %s
    retry:
      max_attempts: 1
      backoff: linear
    circuit_breaker:
      failure_threshold: 3
      reset_timeout: 30s

pipeline:
  - step: %s
  - step: %s
    depends_on: %s
`, pipelineName, agentA, agentB, agentA, agentB, agentA)

	agentStub := func(agentID string) string {
		return fmt.Sprintf(`package agents

import (
	"context"

	"github.com/your-org/fluxroute/pkg/agentfunc"
)

func %s(ctx context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
	select {
	case <-ctx.Done():
		return agentfunc.AgentOutput{}, ctx.Err()
	default:
	}
	return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: input.Payload}, nil
}
`, toFuncName(agentID))
	}

	if err := os.WriteFile(filepath.Join(targetDir, "manifests", "pipeline.yaml"), []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "agents", agentA+".go"), []byte(agentStub(agentA)), 0o644); err != nil {
		return fmt.Errorf("write agent A stub: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "agents", agentB+".go"), []byte(agentStub(agentB)), 0o644); err != nil {
		return fmt.Errorf("write agent B stub: %w", err)
	}

	readme := fmt.Sprintf("# %s pipeline scaffold\n\nRun with:\n\n```bash\nfluxroute-cli run manifests/pipeline.yaml\n```\n", pipelineName)
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte(readme), 0o644); err != nil {
		return fmt.Errorf("write scaffold README: %w", err)
	}

	return nil
}

func toFuncName(agentID string) string {
	parts := strings.Split(agentID, "_")
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	if out == "" {
		return "GeneratedAgent"
	}
	return out
}

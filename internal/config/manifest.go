package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/your-org/fluxroute/internal/security"
	"github.com/your-org/fluxroute/internal/tenant"
	"gopkg.in/yaml.v3"
)

var (
	ErrManifestEmptyAgents   = errors.New("manifest: agents list is empty")
	ErrManifestEmptyPipeline = errors.New("manifest: pipeline is empty")
)

// Manifest is the top-level router manifest file.
type Manifest struct {
	Router   RouterSettings `yaml:"router"`
	Agents   []AgentBinding `yaml:"agents"`
	Pipeline []PipelineStep `yaml:"pipeline"`
}

// RouterSettings configures the runtime engine.
type RouterSettings struct {
	WorkerPoolSize int    `yaml:"worker_pool_size"`
	ChannelBuffer  int    `yaml:"channel_buffer"`
	DefaultTimeout string `yaml:"default_timeout"`
	Namespace      string `yaml:"namespace"`
	RBAC           RBAC   `yaml:"rbac"`
}

// RBAC configures allowed roles per action.
type RBAC struct {
	RunRoles      []string `yaml:"run_roles"`
	ValidateRoles []string `yaml:"validate_roles"`
	ReplayRoles   []string `yaml:"replay_roles"`
	AdminRoles    []string `yaml:"admin_roles"`
}

// AgentBinding declares an agent registration entry.
type AgentBinding struct {
	ID             string               `yaml:"id"`
	Retry          RetryConfig          `yaml:"retry"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// RetryConfig declares retry options for one agent.
type RetryConfig struct {
	MaxAttempts int    `yaml:"max_attempts"`
	Backoff     string `yaml:"backoff"`
}

// CircuitBreakerConfig declares per-agent circuit breaker options.
type CircuitBreakerConfig struct {
	FailureThreshold int    `yaml:"failure_threshold"`
	ResetTimeout     string `yaml:"reset_timeout"`
	ProbeTimeout     string `yaml:"probe_timeout"`
}

// PipelineStep is one node in the execution DAG.
type PipelineStep struct {
	Step      string `yaml:"step"`
	DependsOn string `yaml:"depends_on,omitempty"`
}

// LoadManifest parses and validates a YAML manifest.
func LoadManifest(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("manifest: read %q: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("manifest: unmarshal %q: %w", path, err)
	}

	if err := ValidateManifest(m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// ValidateManifest enforces structural correctness before runtime.
func ValidateManifest(m Manifest) error {
	if len(m.Agents) == 0 {
		return ErrManifestEmptyAgents
	}
	if len(m.Pipeline) == 0 {
		return ErrManifestEmptyPipeline
	}

	if m.Router.Namespace != "" {
		if err := tenant.Validate(m.Router.Namespace); err != nil {
			return fmt.Errorf("manifest: invalid router.namespace: %w", err)
		}
	}
	for _, roles := range [][]string{
		m.Router.RBAC.RunRoles,
		m.Router.RBAC.ValidateRoles,
		m.Router.RBAC.ReplayRoles,
		m.Router.RBAC.AdminRoles,
	} {
		for _, role := range roles {
			if _, err := security.ParseRole(role); err != nil {
				return fmt.Errorf("manifest: invalid rbac role %q: %w", role, err)
			}
		}
	}

	agents := make(map[string]struct{}, len(m.Agents))
	for _, a := range m.Agents {
		if a.ID == "" {
			return errors.New("manifest: agent id is empty")
		}
		if _, exists := agents[a.ID]; exists {
			return fmt.Errorf("manifest: duplicate agent id %q", a.ID)
		}
		agents[a.ID] = struct{}{}

		if a.CircuitBreaker.FailureThreshold < 0 {
			return fmt.Errorf("manifest: agent %q has negative circuit_breaker.failure_threshold", a.ID)
		}
		if a.CircuitBreaker.ResetTimeout != "" {
			if _, err := time.ParseDuration(a.CircuitBreaker.ResetTimeout); err != nil {
				return fmt.Errorf("manifest: agent %q has invalid circuit_breaker.reset_timeout: %w", a.ID, err)
			}
		}
		if a.CircuitBreaker.ProbeTimeout != "" {
			if _, err := time.ParseDuration(a.CircuitBreaker.ProbeTimeout); err != nil {
				return fmt.Errorf("manifest: agent %q has invalid circuit_breaker.probe_timeout: %w", a.ID, err)
			}
		}
	}

	steps := make(map[string]struct{}, len(m.Pipeline))
	for _, p := range m.Pipeline {
		if p.Step == "" {
			return errors.New("manifest: pipeline step is empty")
		}
		if _, exists := steps[p.Step]; exists {
			return fmt.Errorf("manifest: duplicate pipeline step %q", p.Step)
		}
		if _, ok := agents[p.Step]; !ok {
			return fmt.Errorf("manifest: pipeline step %q has no matching agent", p.Step)
		}
		steps[p.Step] = struct{}{}
	}

	for _, p := range m.Pipeline {
		if p.DependsOn == "" {
			continue
		}
		if p.DependsOn == p.Step {
			return fmt.Errorf("manifest: step %q cannot depend on itself", p.Step)
		}
		if _, ok := steps[p.DependsOn]; !ok {
			return fmt.Errorf("manifest: step %q depends on unknown step %q", p.Step, p.DependsOn)
		}
	}

	if _, err := OrderedPipeline(m); err != nil {
		return err
	}
	return nil
}

// OrderedPipeline returns topological order of pipeline steps.
func OrderedPipeline(m Manifest) ([]PipelineStep, error) {
	stepIndex := make(map[string]int, len(m.Pipeline))
	for i, p := range m.Pipeline {
		stepIndex[p.Step] = i
	}

	inDegree := make(map[string]int, len(m.Pipeline))
	children := make(map[string][]string, len(m.Pipeline))
	for _, p := range m.Pipeline {
		if _, ok := inDegree[p.Step]; !ok {
			inDegree[p.Step] = 0
		}
		if p.DependsOn != "" {
			inDegree[p.Step]++
			children[p.DependsOn] = append(children[p.DependsOn], p.Step)
		}
	}

	queue := make([]string, 0, len(m.Pipeline))
	for _, p := range m.Pipeline {
		if inDegree[p.Step] == 0 {
			queue = append(queue, p.Step)
		}
	}

	orderedNames := make([]string, 0, len(m.Pipeline))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		orderedNames = append(orderedNames, curr)

		for _, child := range children[curr] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	if len(orderedNames) != len(m.Pipeline) {
		return nil, errors.New("manifest: cycle detected in pipeline")
	}

	ordered := make([]PipelineStep, 0, len(m.Pipeline))
	for _, name := range orderedNames {
		ordered = append(ordered, m.Pipeline[stepIndex[name]])
	}
	return ordered, nil
}

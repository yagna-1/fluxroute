package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/your-org/fluxroute/internal/security"
	"github.com/your-org/fluxroute/internal/tenant"
	"github.com/your-org/fluxroute/pkg/agentfunc"
)

// FromEnv loads baseline runtime config from environment with safe defaults.
func FromEnv() agentfunc.RouterConfig {
	cfg := agentfunc.RouterConfig{
		WorkerPoolSize: 10,
		ChannelBuffer:  100,
		DefaultTimeout: 30 * time.Second,
		RetryPolicy: agentfunc.RetryPolicy{
			MaxAttempts: 1,
			Backoff:     agentfunc.BackoffLinear,
		},
		CircuitBreaker: agentfunc.CircuitBreakerPolicy{
			FailureThreshold: 5,
			ResetTimeout:     60 * time.Second,
		},
	}

	if v := os.Getenv("WORKER_POOL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.WorkerPoolSize = n
		}
	}
	if v := os.Getenv("CHANNEL_BUFFER"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ChannelBuffer = n
		}
	}
	if v := os.Getenv("DEFAULT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.DefaultTimeout = d
		}
	}
	if v := os.Getenv("CIRCUIT_FAILURE_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.CircuitBreaker.FailureThreshold = n
		}
	}
	if v := os.Getenv("CIRCUIT_RESET_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.CircuitBreaker.ResetTimeout = d
		}
	}

	return cfg
}

// RouterConfigFromManifest merges manifest router settings on top of a base config.
func RouterConfigFromManifest(m Manifest, base agentfunc.RouterConfig) (agentfunc.RouterConfig, error) {
	cfg := base
	if m.Router.WorkerPoolSize > 0 {
		cfg.WorkerPoolSize = m.Router.WorkerPoolSize
	}
	if m.Router.ChannelBuffer > 0 {
		cfg.ChannelBuffer = m.Router.ChannelBuffer
	}
	if m.Router.DefaultTimeout != "" {
		d, err := time.ParseDuration(m.Router.DefaultTimeout)
		if err != nil {
			return agentfunc.RouterConfig{}, fmt.Errorf("manifest: invalid router.default_timeout: %w", err)
		}
		cfg.DefaultTimeout = d
	}
	return cfg, nil
}

// RetryPolicyFromConfig converts manifest retry settings into runtime policy.
func RetryPolicyFromConfig(rc RetryConfig) agentfunc.RetryPolicy {
	p := agentfunc.RetryPolicy{
		MaxAttempts: rc.MaxAttempts,
		Backoff:     parseBackoff(rc.Backoff),
	}
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = 1
	}
	return p
}

// CircuitBreakerPolicyFromConfig converts manifest circuit breaker settings into runtime policy.
func CircuitBreakerPolicyFromConfig(cbc CircuitBreakerConfig, fallback agentfunc.CircuitBreakerPolicy) (agentfunc.CircuitBreakerPolicy, error) {
	p := fallback
	if cbc.FailureThreshold > 0 {
		p.FailureThreshold = cbc.FailureThreshold
	}
	if cbc.ResetTimeout != "" {
		d, err := time.ParseDuration(cbc.ResetTimeout)
		if err != nil {
			return agentfunc.CircuitBreakerPolicy{}, fmt.Errorf("invalid circuit breaker reset_timeout: %w", err)
		}
		p.ResetTimeout = d
	}
	if p.FailureThreshold < 0 {
		p.FailureThreshold = 0
	}
	if p.ResetTimeout <= 0 {
		p.ResetTimeout = 60 * time.Second
	}
	return p, nil
}

func parseBackoff(v string) agentfunc.BackoffStrategy {
	switch v {
	case string(agentfunc.BackoffExponential):
		return agentfunc.BackoffExponential
	case string(agentfunc.BackoffExponentialJitter):
		return agentfunc.BackoffExponentialJitter
	default:
		return agentfunc.BackoffLinear
	}
}

// NamespaceFromManifest returns a validated normalized namespace.
func NamespaceFromManifest(m Manifest) (string, error) {
	ns := tenant.Normalize(m.Router.Namespace)
	if err := tenant.Validate(ns); err != nil {
		return "", err
	}
	return ns, nil
}

// RBACPolicyFromManifest converts manifest RBAC config to runtime policy.
func RBACPolicyFromManifest(m Manifest) (security.Policy, error) {
	parseOrDefault := func(values []string, fallback []security.Role) ([]security.Role, error) {
		if len(values) == 0 {
			return fallback, nil
		}
		return security.ParseRoles(values)
	}

	runRoles, err := parseOrDefault(m.Router.RBAC.RunRoles, []security.Role{security.RoleOperator, security.RoleAdmin})
	if err != nil {
		return security.Policy{}, err
	}
	validateRoles, err := parseOrDefault(m.Router.RBAC.ValidateRoles, []security.Role{security.RoleViewer, security.RoleOperator, security.RoleAdmin})
	if err != nil {
		return security.Policy{}, err
	}
	replayRoles, err := parseOrDefault(m.Router.RBAC.ReplayRoles, []security.Role{security.RoleOperator, security.RoleAdmin})
	if err != nil {
		return security.Policy{}, err
	}
	adminRoles, err := parseOrDefault(m.Router.RBAC.AdminRoles, []security.Role{security.RoleAdmin})
	if err != nil {
		return security.Policy{}, err
	}

	return security.NewPolicy(map[security.Action][]security.Role{
		security.ActionRun:      runRoles,
		security.ActionValidate: validateRoles,
		security.ActionReplay:   replayRoles,
		security.ActionAdmin:    adminRoles,
	}), nil
}

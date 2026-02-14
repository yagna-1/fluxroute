package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/your-org/agent-router/pkg/agentfunc"
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

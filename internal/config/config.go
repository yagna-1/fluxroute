package config

import (
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

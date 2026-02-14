package config

import (
	"os"
	"time"

	"github.com/your-org/agent-router/pkg/agentfunc"
)

// FromEnv loads baseline runtime config from environment with safe defaults.
func FromEnv() agentfunc.RouterConfig {
	_ = os.Getenv("WORKER_POOL_SIZE")
	_ = os.Getenv("CHANNEL_BUFFER")
	_ = os.Getenv("DEFAULT_TIMEOUT")

	return agentfunc.RouterConfig{
		WorkerPoolSize: 10,
		ChannelBuffer:  100,
		DefaultTimeout: 30 * time.Second,
		RetryPolicy: agentfunc.RetryPolicy{
			MaxAttempts: 1,
			Backoff:     agentfunc.BackoffLinear,
		},
	}
}

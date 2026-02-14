package agentfunc

import (
	"context"
	"time"
)

// AgentFunc is the universal contract for all agents.
type AgentFunc func(ctx context.Context, input AgentInput) (AgentOutput, error)

// AgentInput is the request payload for an agent invocation.
type AgentInput struct {
	TaskID    string
	RequestID string
	Payload   []byte
	Metadata  map[string]string
	Timestamp time.Time
}

// AgentOutput is the response payload from an agent invocation.
type AgentOutput struct {
	RequestID string
	Payload   []byte
	Metadata  map[string]string
	Duration  time.Duration
}

// BackoffStrategy defines retry wait behavior.
type BackoffStrategy string

const (
	BackoffLinear            BackoffStrategy = "linear"
	BackoffExponential       BackoffStrategy = "exponential"
	BackoffExponentialJitter BackoffStrategy = "exponential_jitter"
)

// RetryPolicy configures retry behavior.
type RetryPolicy struct {
	MaxAttempts   int
	Backoff       BackoffStrategy
	RetryableErrs []error
}

// CircuitBreakerPolicy configures failure threshold and reset behavior.
type CircuitBreakerPolicy struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	ProbeTimeout     time.Duration
}

// RouterConfig is top-level runtime configuration.
type RouterConfig struct {
	WorkerPoolSize int
	ChannelBuffer  int
	DefaultTimeout time.Duration
	RetryPolicy    RetryPolicy
	CircuitBreaker CircuitBreakerPolicy
}

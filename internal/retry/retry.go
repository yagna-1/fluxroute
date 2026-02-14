package retry

import (
	"context"
	"time"

	"github.com/your-org/agent-router/pkg/agentfunc"
)

// Execute runs fn with a simple configurable retry policy.
func Execute(ctx context.Context, policy agentfunc.RetryPolicy, fn func(context.Context) error) error {
	attempts := policy.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for i := 1; i <= attempts; i++ {
		if err := fn(ctx); err != nil {
			lastErr = err
			if i == attempts {
				return lastErr
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(BackoffDuration(policy.Backoff, i)):
			}
			continue
		}
		return nil
	}
	return lastErr
}

// BackoffDuration returns the sleep interval for a given retry attempt.
func BackoffDuration(strategy agentfunc.BackoffStrategy, attempt int) time.Duration {
	base := 100 * time.Millisecond
	switch strategy {
	case agentfunc.BackoffExponential:
		return base * time.Duration(1<<uint(attempt-1))
	case agentfunc.BackoffExponentialJitter:
		exp := base * time.Duration(1<<uint(attempt-1))
		// Deterministic jitter keeps replay behavior stable while spreading retries.
		jitter := time.Duration((attempt*37)%100) * time.Millisecond
		return exp + jitter
	default:
		return base * time.Duration(attempt)
	}
}

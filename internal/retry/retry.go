package retry

import (
	"context"
	"math/rand"
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
			case <-time.After(backoffFor(policy.Backoff, i)):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func backoffFor(strategy agentfunc.BackoffStrategy, attempt int) time.Duration {
	base := 100 * time.Millisecond
	switch strategy {
	case agentfunc.BackoffExponential:
		return base * time.Duration(1<<uint(attempt-1))
	case agentfunc.BackoffExponentialJitter:
		exp := base * time.Duration(1<<uint(attempt-1))
		jitter := time.Duration(rand.Int63n(int64(base)))
		return exp + jitter
	default:
		return base * time.Duration(attempt)
	}
}

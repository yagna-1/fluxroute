package retry

import (
	"errors"
	"fmt"
)

var (
	ErrAgentTimeout   = errors.New("agent timeout")
	ErrCircuitOpen    = errors.New("circuit breaker open")
	ErrAgentPanic     = errors.New("agent panicked")
	ErrInvalidPayload = errors.New("invalid agent payload")
)

// AgentError wraps an underlying error with retryability metadata.
type AgentError struct {
	Cause     error
	Retryable bool
}

func (e AgentError) Error() string {
	if e.Cause == nil {
		return "agent error"
	}
	return fmt.Sprintf("agent error: %v", e.Cause)
}

func (e AgentError) Unwrap() error {
	return e.Cause
}

// NonRetryable marks an error as not eligible for retries.
func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return AgentError{Cause: err, Retryable: false}
}

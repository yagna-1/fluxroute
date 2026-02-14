package trace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/your-org/agent-router/pkg/agentfunc"
)

// ResolveAgentFn resolves agent IDs to runtime implementations.
type ResolveAgentFn func(agentID string) (agentfunc.AgentFunc, bool)

// ReplayAndCompare re-executes final recorded invocations and validates output equality and order.
func ReplayAndCompare(ctx context.Context, tr ExecutionTrace, timeout time.Duration, resolve ResolveAgentFn) error {
	if len(tr.Steps) == 0 {
		return errors.New("trace replay: no steps to replay")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	expectedByInvocation := make(map[string]Step)
	for _, s := range tr.Steps {
		prev, ok := expectedByInvocation[s.InvocationID]
		if !ok || s.Attempt >= prev.Attempt {
			expectedByInvocation[s.InvocationID] = s
		}
	}

	invocationIDs := make([]string, 0, len(expectedByInvocation))
	for id := range expectedByInvocation {
		invocationIDs = append(invocationIDs, id)
	}
	sort.Strings(invocationIDs)

	for _, invID := range invocationIDs {
		expected := expectedByInvocation[invID]
		fn, ok := resolve(expected.AgentID)
		if !ok {
			return fmt.Errorf("trace replay: agent not found: %s", expected.AgentID)
		}

		runCtx, cancel := context.WithTimeout(ctx, timeout)
		actualOut, actualErr := safeCall(fn, runCtx, expected.Input)
		cancel()

		if expected.Error != "" {
			if actualErr == nil {
				return fmt.Errorf("trace replay: invocation %s expected error %q but got nil", invID, expected.Error)
			}
			if actualErr.Error() != expected.Error {
				return fmt.Errorf("trace replay: invocation %s error mismatch: got %q want %q", invID, actualErr.Error(), expected.Error)
			}
			continue
		}

		if actualErr != nil {
			return fmt.Errorf("trace replay: invocation %s unexpected error: %v", invID, actualErr)
		}
		if actualOut.RequestID != expected.Output.RequestID {
			return fmt.Errorf("trace replay: invocation %s request_id mismatch: got %q want %q", invID, actualOut.RequestID, expected.Output.RequestID)
		}
		if !bytes.Equal(actualOut.Payload, expected.Output.Payload) {
			return fmt.Errorf("trace replay: invocation %s payload mismatch", invID)
		}
	}

	return nil
}

func safeCall(fn agentfunc.AgentFunc, ctx context.Context, in agentfunc.AgentInput) (out agentfunc.AgentOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("agent panic: %v", r)
		}
	}()
	return fn(ctx, in)
}

package trace

import (
	"time"

	"github.com/your-org/fluxroute/pkg/agentfunc"
)

// ExecutionTrace captures the full run for replay/debug.
type ExecutionTrace struct {
	TaskID       string
	Steps        []Step
	StartTime    time.Time
	EndTime      time.Time
	TotalLatency time.Duration
}

// Step is a single agent invocation record.
type Step struct {
	InvocationID string
	AgentID      string
	RequestID    string
	Input        agentfunc.AgentInput
	Output       agentfunc.AgentOutput
	Error        string
	Duration     time.Duration
	Attempt      int
}

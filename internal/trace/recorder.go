package trace

import (
	"sort"
	"sync"
	"time"

	"github.com/your-org/fluxroute/pkg/agentfunc"
)

// Recorder captures per-attempt trace steps and finalizes deterministic order.
type Recorder struct {
	mu    sync.Mutex
	trace ExecutionTrace
}

func NewRecorder(taskID string, start time.Time) *Recorder {
	return &Recorder{trace: ExecutionTrace{TaskID: taskID, StartTime: start}}
}

func (r *Recorder) AddStep(step Step) {
	r.mu.Lock()
	defer r.mu.Unlock()

	step.Input = cloneInput(step.Input)
	step.Output = cloneOutput(step.Output)
	r.trace.Steps = append(r.trace.Steps, step)
}

func (r *Recorder) Finalize(end time.Time) ExecutionTrace {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := ExecutionTrace{
		TaskID:       r.trace.TaskID,
		StartTime:    r.trace.StartTime,
		EndTime:      end,
		TotalLatency: end.Sub(r.trace.StartTime),
		Steps:        append([]Step(nil), r.trace.Steps...),
	}

	sort.Slice(out.Steps, func(i, j int) bool {
		if out.Steps[i].InvocationID != out.Steps[j].InvocationID {
			return out.Steps[i].InvocationID < out.Steps[j].InvocationID
		}
		if out.Steps[i].Attempt != out.Steps[j].Attempt {
			return out.Steps[i].Attempt < out.Steps[j].Attempt
		}
		return out.Steps[i].RequestID < out.Steps[j].RequestID
	})
	return out
}

func cloneInput(in agentfunc.AgentInput) agentfunc.AgentInput {
	out := in
	if in.Payload != nil {
		out.Payload = append([]byte(nil), in.Payload...)
	}
	if in.Metadata != nil {
		out.Metadata = make(map[string]string, len(in.Metadata))
		for k, v := range in.Metadata {
			out.Metadata[k] = v
		}
	}
	return out
}

func cloneOutput(in agentfunc.AgentOutput) agentfunc.AgentOutput {
	out := in
	if in.Payload != nil {
		out.Payload = append([]byte(nil), in.Payload...)
	}
	if in.Metadata != nil {
		out.Metadata = make(map[string]string, len(in.Metadata))
		for k, v := range in.Metadata {
			out.Metadata[k] = v
		}
	}
	return out
}

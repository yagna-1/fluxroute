package trace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

// Divergence describes where two traces first diverge.
type Divergence struct {
	InvocationID string
	Field        string
	Expected     string
	Actual       string
}

// Compare traces and return divergence list. Empty list means equivalent replay-significant behavior.
func Compare(expected ExecutionTrace, actual ExecutionTrace) []Divergence {
	expMap := latestByInvocation(expected)
	actMap := latestByInvocation(actual)

	ids := make([]string, 0, len(expMap)+len(actMap))
	seen := map[string]struct{}{}
	for id := range expMap {
		ids = append(ids, id)
		seen[id] = struct{}{}
	}
	for id := range actMap {
		if _, ok := seen[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	out := make([]Divergence, 0)
	for _, id := range ids {
		e, eok := expMap[id]
		a, aok := actMap[id]
		if !eok {
			out = append(out, Divergence{InvocationID: id, Field: "missing_expected", Actual: a.AgentID})
			continue
		}
		if !aok {
			out = append(out, Divergence{InvocationID: id, Field: "missing_actual", Expected: e.AgentID})
			continue
		}
		if e.AgentID != a.AgentID {
			out = append(out, Divergence{InvocationID: id, Field: "agent_id", Expected: e.AgentID, Actual: a.AgentID})
		}
		if e.Error != a.Error {
			out = append(out, Divergence{InvocationID: id, Field: "error", Expected: e.Error, Actual: a.Error})
		}
		if e.Output.RequestID != a.Output.RequestID {
			out = append(out, Divergence{InvocationID: id, Field: "request_id", Expected: e.Output.RequestID, Actual: a.Output.RequestID})
		}
		ep := payloadHash(e.Output.Payload)
		ap := payloadHash(a.Output.Payload)
		if ep != ap {
			out = append(out, Divergence{InvocationID: id, Field: "payload_hash", Expected: ep, Actual: ap})
		}
	}
	return out
}

func FormatDivergence(div []Divergence) string {
	if len(div) == 0 {
		return "no divergence detected"
	}
	msg := "trace divergence detected:\n"
	for _, d := range div {
		msg += fmt.Sprintf("- invocation=%s field=%s expected=%q actual=%q\n", d.InvocationID, d.Field, d.Expected, d.Actual)
	}
	return msg
}

func latestByInvocation(tr ExecutionTrace) map[string]Step {
	m := make(map[string]Step)
	for _, s := range tr.Steps {
		prev, ok := m[s.InvocationID]
		if !ok || s.Attempt >= prev.Attempt {
			m[s.InvocationID] = s
		}
	}
	return m
}

func payloadHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

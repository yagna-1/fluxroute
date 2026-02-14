package unit

import (
	"testing"
	"time"

	"github.com/your-org/fluxroute/internal/metrics"
)

func TestInMemoryRecorderSnapshot(t *testing.T) {
	r := metrics.NewInMemoryRecorder()
	r.ObserveInvocation("a", "success", 10*time.Millisecond)
	r.ObserveRetry("a")
	r.ObserveInvocation("a", "error", 5*time.Millisecond)

	s := r.Snapshot()
	if s.TotalInvocations != 2 {
		t.Fatalf("total invocations mismatch: %d", s.TotalInvocations)
	}
	if s.ErrorInvocations != 1 {
		t.Fatalf("error invocations mismatch: %d", s.ErrorInvocations)
	}
	if s.RetryAttempts != 1 {
		t.Fatalf("retry attempts mismatch: %d", s.RetryAttempts)
	}
	if s.ByAgent["a"].Successes != 1 || s.ByAgent["a"].Errors != 1 || s.ByAgent["a"].Retries != 1 {
		t.Fatalf("unexpected agent stats: %+v", s.ByAgent["a"])
	}
}

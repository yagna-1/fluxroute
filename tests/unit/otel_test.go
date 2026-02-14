package unit

import (
	"context"
	"os"
	"testing"

	"github.com/your-org/agent-router/internal/trace"
)

func TestSetupOTelFromEnv(t *testing.T) {
	oldEnabled := os.Getenv("TRACE_ENABLED")
	oldEndpoint := os.Getenv("TRACE_ENDPOINT")
	t.Cleanup(func() {
		_ = os.Setenv("TRACE_ENABLED", oldEnabled)
		_ = os.Setenv("TRACE_ENDPOINT", oldEndpoint)
	})

	_ = os.Setenv("TRACE_ENABLED", "true")
	_ = os.Setenv("TRACE_ENDPOINT", "")

	rt, err := trace.SetupOTelFromEnv("agent-router-test")
	if err != nil {
		t.Fatalf("setup otel: %v", err)
	}
	if rt.Tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown otel: %v", err)
	}
}

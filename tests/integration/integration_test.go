package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/your-org/fluxroute/internal/app"
)

func TestRunManifestEndToEndSuccess(t *testing.T) {
	manifestPath := writeManifest(t, `
router:
  worker_pool_size: 4
  channel_buffer: 10
  default_timeout: 5s
agents:
  - id: flaky_summarize
    retry:
      max_attempts: 2
      backoff: linear
  - id: classify_agent
    retry:
      max_attempts: 1
      backoff: linear
pipeline:
  - step: flaky_summarize
  - step: classify_agent
    depends_on: flaky_summarize
`)

	tracePath := filepath.Join(t.TempDir(), "trace.json")
	oldTrace := os.Getenv("TRACE_OUTPUT")
	t.Cleanup(func() { _ = os.Setenv("TRACE_OUTPUT", oldTrace) })
	_ = os.Setenv("TRACE_OUTPUT", tracePath)

	var out bytes.Buffer
	if err := app.RunManifest(manifestPath, &out); err != nil {
		t.Fatalf("run manifest failed: %v", err)
	}

	stdout := out.String()
	if !strings.Contains(stdout, "router executed 2 invocation(s)") {
		t.Fatalf("unexpected output: %s", stdout)
	}
	if _, err := os.Stat(tracePath); err != nil {
		t.Fatalf("trace file not written: %v", err)
	}
}

func TestRunManifestReturnsErrorOnFailedPipeline(t *testing.T) {
	manifestPath := writeManifest(t, `
router:
  worker_pool_size: 2
  channel_buffer: 10
  default_timeout: 5s
agents:
  - id: fail_primary
    retry:
      max_attempts: 1
      backoff: linear
  - id: child_agent
    retry:
      max_attempts: 1
      backoff: linear
pipeline:
  - step: fail_primary
  - step: child_agent
    depends_on: fail_primary
`)

	var out bytes.Buffer
	err := app.RunManifest(manifestPath, &out)
	if err == nil {
		t.Fatalf("expected pipeline error, stdout=%s", out.String())
	}
	if !strings.Contains(err.Error(), "failed invocation") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunManifestParallelIndependentSteps(t *testing.T) {
	manifestPath := writeManifest(t, `
router:
  worker_pool_size: 4
  channel_buffer: 10
  default_timeout: 1s
agents:
  - id: slow_a
    retry:
      max_attempts: 1
      backoff: linear
  - id: slow_b
    retry:
      max_attempts: 1
      backoff: linear
pipeline:
  - step: slow_a
  - step: slow_b
`)

	var out bytes.Buffer
	start := time.Now()
	if err := app.RunManifest(manifestPath, &out); err != nil {
		t.Fatalf("run manifest failed: %v", err)
	}
	elapsed := time.Since(start)

	// Each slow_* agent sleeps for ~200ms. Independent steps should run in parallel.
	if elapsed >= 350*time.Millisecond {
		t.Fatalf("expected parallel execution under 350ms, got %s (output=%s)", elapsed, out.String())
	}
}

func TestRunManifestTimeoutPropagation(t *testing.T) {
	manifestPath := writeManifest(t, `
router:
  worker_pool_size: 2
  channel_buffer: 10
  default_timeout: 20ms
agents:
  - id: slow_timeout_agent
    retry:
      max_attempts: 1
      backoff: linear
pipeline:
  - step: slow_timeout_agent
`)

	var out bytes.Buffer
	err := app.RunManifest(manifestPath, &out)
	if err == nil {
		t.Fatalf("expected timeout error, stdout=%s", out.String())
	}
	if !strings.Contains(err.Error(), "failed invocation") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func writeManifest(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

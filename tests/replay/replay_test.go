package replay

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/fluxroute/internal/app"
)

func TestReplayFromTraceFile(t *testing.T) {
	manifestPath := writeManifest(t, `
router:
  worker_pool_size: 4
  channel_buffer: 10
  default_timeout: 5s
agents:
  - id: summarize_agent
    retry:
      max_attempts: 1
      backoff: linear
  - id: classify_agent
    retry:
      max_attempts: 1
      backoff: linear
pipeline:
  - step: summarize_agent
  - step: classify_agent
    depends_on: summarize_agent
`)

	tracePath := filepath.Join(t.TempDir(), "trace.json")
	oldTrace := os.Getenv("TRACE_OUTPUT")
	t.Cleanup(func() { _ = os.Setenv("TRACE_OUTPUT", oldTrace) })
	_ = os.Setenv("TRACE_OUTPUT", tracePath)

	if _, err := app.RunManifestReport(manifestPath); err != nil {
		t.Fatalf("run manifest report: %v", err)
	}

	var out bytes.Buffer
	if err := app.ReplayTrace(tracePath, &out); err != nil {
		t.Fatalf("replay trace failed: %v", err)
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

package unit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/fluxroute/internal/app"
)

func TestAppValidateManifest(t *testing.T) {
	path := writeManifest(t, `
router:
  worker_pool_size: 10
  channel_buffer: 20
  default_timeout: 30s
agents:
  - id: summarize_agent
  - id: classify_agent
pipeline:
  - step: summarize_agent
  - step: classify_agent
    depends_on: summarize_agent
`)

	if err := app.ValidateManifest(path); err != nil {
		t.Fatalf("expected valid manifest, got %v", err)
	}
}

func TestRunManifest(t *testing.T) {
	path := writeManifest(t, `
router:
  worker_pool_size: 10
  channel_buffer: 20
  default_timeout: 30s
agents:
  - id: summarize_agent
  - id: classify_agent
pipeline:
  - step: summarize_agent
  - step: classify_agent
    depends_on: summarize_agent
`)

	var out bytes.Buffer
	if err := app.RunManifest(path, &out); err != nil {
		t.Fatalf("run manifest failed: %v", err)
	}

	stdout := out.String()
	if !strings.Contains(stdout, "router executed 2 invocation(s)") {
		t.Fatalf("unexpected output: %s", stdout)
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

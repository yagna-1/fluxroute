package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLIHelp(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runCLI([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Examples:") {
		t.Fatalf("expected help output with examples, got %q", out.String())
	}
}

func TestRunCLIUnknownCommandJSON(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runCLI([]string{"--json", "does-not-exist"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), `"status":"error"`) {
		t.Fatalf("expected json error output, got %q", errOut.String())
	}
}

func TestRunCLIValidateJSON(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.yaml")
	manifest := `
router:
  worker_pool_size: 10
  channel_buffer: 20
  default_timeout: 30s
agents:
  - id: summarize_agent
pipeline:
  - step: summarize_agent
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runCLI([]string{"--json", "validate", manifestPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"status":"ok"`) || !strings.Contains(out.String(), `"valid":true`) {
		t.Fatalf("expected success json payload, got %q", out.String())
	}
}

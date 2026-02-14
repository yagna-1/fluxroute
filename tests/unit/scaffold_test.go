package unit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/agent-router/internal/app"
)

func TestScaffoldProject(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	var out bytes.Buffer
	if err := app.ScaffoldProject(target, "demo", &out); err != nil {
		t.Fatalf("scaffold project failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "manifests", "pipeline.yaml")); err != nil {
		t.Fatalf("expected manifest file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "agents", "demo_step_a.go")); err != nil {
		t.Fatalf("expected agent stub: %v", err)
	}
}

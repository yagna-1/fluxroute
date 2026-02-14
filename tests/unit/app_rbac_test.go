package unit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/fluxroute/internal/app"
)

func TestRunManifestRBACDeniedForViewer(t *testing.T) {
	manifestPath := writeRBACManifest(t)
	oldRole := os.Getenv("REQUEST_ROLE")
	t.Cleanup(func() { _ = os.Setenv("REQUEST_ROLE", oldRole) })
	_ = os.Setenv("REQUEST_ROLE", "viewer")

	var out bytes.Buffer
	err := app.RunManifest(manifestPath, &out)
	if err == nil {
		t.Fatal("expected rbac denied error")
	}
	if !strings.Contains(err.Error(), "rbac denied") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeRBACManifest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	data := `
router:
  worker_pool_size: 2
  channel_buffer: 8
  default_timeout: 5s
  namespace: test
  rbac:
    run_roles: [operator, admin]
    validate_roles: [viewer, operator, admin]
    replay_roles: [operator, admin]
agents:
  - id: summarize_agent
pipeline:
  - step: summarize_agent
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

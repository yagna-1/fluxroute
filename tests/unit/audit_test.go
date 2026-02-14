package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/agent-router/internal/audit"
)

func TestAuditWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l := audit.NewLogger(path)
	if err := l.Write("operator", "run_manifest", "manifest.yaml", "success", nil); err != nil {
		t.Fatalf("audit write failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected audit log content")
	}
}

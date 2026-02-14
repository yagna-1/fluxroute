package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/fluxroute/internal/audit"
)

func TestAuditExportJSONLToCSV(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "audit.log")
	outPath := filepath.Join(dir, "audit.csv")

	l := audit.NewLogger(inPath)
	if err := l.Write("admin", "run_manifest", "manifest.yaml", "success", nil); err != nil {
		t.Fatalf("write audit log: %v", err)
	}

	if err := audit.ExportJSONLToCSV(inPath, outPath); err != nil {
		t.Fatalf("export audit csv: %v", err)
	}

	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected csv output")
	}
}

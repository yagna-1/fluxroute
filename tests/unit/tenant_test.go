package unit

import (
	"testing"

	"github.com/your-org/agent-router/internal/tenant"
)

func TestNamespaceNormalizeValidate(t *testing.T) {
	ns := tenant.Normalize("")
	if ns != "default" {
		t.Fatalf("expected default namespace, got %q", ns)
	}
	if err := tenant.Validate("team_a1"); err != nil {
		t.Fatalf("expected valid namespace, got %v", err)
	}
	if err := tenant.Validate("Bad Namespace"); err == nil {
		t.Fatal("expected invalid namespace error")
	}
}

package unit

import (
	"testing"

	"github.com/your-org/fluxroute/internal/security"
)

func TestRBACPolicy(t *testing.T) {
	p := security.DefaultPolicy()
	if !p.IsAllowed(security.RoleOperator, security.ActionRun) {
		t.Fatal("operator should run")
	}
	if p.IsAllowed(security.RoleViewer, security.ActionRun) {
		t.Fatal("viewer should not run")
	}
}

func TestParseRole(t *testing.T) {
	if _, err := security.ParseRole("admin"); err != nil {
		t.Fatalf("parse admin: %v", err)
	}
	if _, err := security.ParseRole("invalid"); err == nil {
		t.Fatal("expected parse error for invalid role")
	}
}

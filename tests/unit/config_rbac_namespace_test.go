package unit

import (
	"testing"

	"github.com/your-org/agent-router/internal/config"
	"github.com/your-org/agent-router/internal/security"
)

func TestRBACPolicyFromManifest(t *testing.T) {
	m := config.Manifest{Router: config.RouterSettings{RBAC: config.RBAC{RunRoles: []string{"admin"}}}}
	p, err := config.RBACPolicyFromManifest(m)
	if err != nil {
		t.Fatalf("build policy: %v", err)
	}
	if p.IsAllowed(security.RoleOperator, security.ActionRun) {
		t.Fatal("operator should not be allowed when run_roles=[admin]")
	}
	if !p.IsAllowed(security.RoleAdmin, security.ActionRun) {
		t.Fatal("admin should be allowed")
	}
}

func TestNamespaceFromManifest(t *testing.T) {
	m := config.Manifest{Router: config.RouterSettings{Namespace: "Team_A"}}
	ns, err := config.NamespaceFromManifest(m)
	if err != nil {
		t.Fatalf("namespace parse failed: %v", err)
	}
	if ns != "team_a" {
		t.Fatalf("expected normalized namespace team_a, got %q", ns)
	}
}

package security

import (
	"fmt"
	"strings"
)

type Action string

const (
	ActionRun      Action = "run_manifest"
	ActionValidate Action = "validate_manifest"
	ActionReplay   Action = "replay_trace"
	ActionAdmin    Action = "admin"
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
	RoleAdmin    Role = "admin"
)

func (r Role) String() string {
	return string(r)
}

// Policy maps actions to allowed roles.
type Policy struct {
	allowed map[Action]map[Role]struct{}
}

func DefaultPolicy() Policy {
	return NewPolicy(map[Action][]Role{
		ActionRun:      {RoleOperator, RoleAdmin},
		ActionValidate: {RoleViewer, RoleOperator, RoleAdmin},
		ActionReplay:   {RoleOperator, RoleAdmin},
		ActionAdmin:    {RoleAdmin},
	})
}

func NewPolicy(allowed map[Action][]Role) Policy {
	out := Policy{allowed: make(map[Action]map[Role]struct{}, len(allowed))}
	for act, roles := range allowed {
		set := make(map[Role]struct{}, len(roles))
		for _, r := range roles {
			set[r] = struct{}{}
		}
		out.allowed[act] = set
	}
	return out
}

func (p Policy) IsAllowed(role Role, action Action) bool {
	set, ok := p.allowed[action]
	if !ok {
		return false
	}
	_, ok = set[role]
	return ok
}

func ParseRole(raw string) (Role, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch Role(s) {
	case RoleViewer, RoleOperator, RoleAdmin:
		return Role(s), nil
	default:
		return "", fmt.Errorf("unknown role: %q", raw)
	}
}

func ParseRoles(raw []string) ([]Role, error) {
	roles := make([]Role, 0, len(raw))
	for _, r := range raw {
		role, err := ParseRole(r)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

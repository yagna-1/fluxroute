package agent

import (
	"errors"
	"fmt"
	"sync"

	"github.com/your-org/agent-router/pkg/agentfunc"
)

var (
	ErrEmptyAgentID     = errors.New("agent id is empty")
	ErrNilAgentFunc     = errors.New("agent func is nil")
	ErrDuplicateAgentID = errors.New("agent id already registered")
)

// Registry stores agent implementations by ID.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]agentfunc.AgentFunc
}

func NewRegistry() *Registry {
	return &Registry{agents: make(map[string]agentfunc.AgentFunc)}
}

func (r *Registry) Register(agentID string, fn agentfunc.AgentFunc) error {
	return r.RegisterVersion(agentID, "v1", fn)
}

// RegisterVersion stores an agent implementation under id@version.
func (r *Registry) RegisterVersion(agentID string, version string, fn agentfunc.AgentFunc) error {
	if agentID == "" {
		return ErrEmptyAgentID
	}
	if version == "" {
		version = "v1"
	}
	if fn == nil {
		return ErrNilAgentFunc
	}

	key := VersionedID(agentID, version)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[key]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateAgentID, key)
	}
	r.agents[key] = fn
	return nil
}

func (r *Registry) Get(agentID string) (agentfunc.AgentFunc, bool) {
	return r.GetVersion(agentID, "v1")
}

// GetVersion returns an agent implementation from id@version.
func (r *Registry) GetVersion(agentID string, version string) (agentfunc.AgentFunc, bool) {
	if version == "" {
		version = "v1"
	}
	key := VersionedID(agentID, version)

	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.agents[key]
	return fn, ok
}

// VersionedID builds the normalized agent identifier form "id@version".
func VersionedID(agentID string, version string) string {
	if version == "" {
		version = "v1"
	}
	return agentID + "@" + version
}

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
	if agentID == "" {
		return ErrEmptyAgentID
	}
	if fn == nil {
		return ErrNilAgentFunc
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateAgentID, agentID)
	}
	r.agents[agentID] = fn
	return nil
}

func (r *Registry) Get(agentID string) (agentfunc.AgentFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.agents[agentID]
	return fn, ok
}

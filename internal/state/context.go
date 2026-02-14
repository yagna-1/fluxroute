package state

import "context"

type contextKey string

const stateKey contextKey = "fluxroute_state"

// Snapshot is immutable request-scoped state.
type Snapshot struct {
	TaskID   string
	Metadata map[string]string
}

func ToContext(ctx context.Context, s Snapshot) context.Context {
	return context.WithValue(ctx, stateKey, s)
}

func FromContext(ctx context.Context) (Snapshot, bool) {
	s, ok := ctx.Value(stateKey).(Snapshot)
	return s, ok
}

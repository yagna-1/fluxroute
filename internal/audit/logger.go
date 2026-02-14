package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is one audit-log record.
type Event struct {
	Timestamp string `json:"ts"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// Logger writes JSONL audit records.
type Logger struct {
	mu   sync.Mutex
	path string
}

func NewLogger(path string) *Logger {
	return &Logger{path: path}
}

func (l *Logger) Enabled() bool {
	return l != nil && l.path != ""
}

func (l *Logger) Write(actor, action, resource, status string, err error) error {
	if !l.Enabled() {
		return nil
	}

	ev := Event{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		Status:    status,
	}
	if err != nil {
		ev.Error = err.Error()
	}
	b, mErr := json.Marshal(ev)
	if mErr != nil {
		return fmt.Errorf("audit marshal: %w", mErr)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if mkErr := os.MkdirAll(filepath.Dir(l.path), 0o755); mkErr != nil {
		return fmt.Errorf("audit mkdir: %w", mkErr)
	}
	f, openErr := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if openErr != nil {
		return fmt.Errorf("audit open: %w", openErr)
	}
	defer func() { _ = f.Close() }()

	if _, wErr := f.Write(append(b, '\n')); wErr != nil {
		return fmt.Errorf("audit write: %w", wErr)
	}
	return nil
}

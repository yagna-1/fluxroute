package coordinator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Lease interface {
	Release(context.Context) error
}

type Coordinator interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lease, error)
}

type memoryCoordinator struct {
	mu    sync.Mutex
	locks map[string]time.Time
}

type memoryLease struct {
	key string
	c   *memoryCoordinator
}

func NewMemoryCoordinator() Coordinator {
	return &memoryCoordinator{locks: make(map[string]time.Time)}
}

func (c *memoryCoordinator) Acquire(ctx context.Context, key string, ttl time.Duration) (Lease, error) {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	for {
		c.mu.Lock()
		exp, exists := c.locks[key]
		now := time.Now()
		if !exists || now.After(exp) {
			c.locks[key] = now.Add(ttl)
			c.mu.Unlock()
			return &memoryLease{key: key, c: c}, nil
		}
		c.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("acquire lease: %w", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func (l *memoryLease) Release(_ context.Context) error {
	l.c.mu.Lock()
	defer l.c.mu.Unlock()
	delete(l.c.locks, l.key)
	return nil
}

type fileCoordinator struct {
	dir string
}

type fileLease struct {
	path string
}

func NewFileCoordinator(dir string) Coordinator {
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "fluxroute-coordination")
	}
	return &fileCoordinator{dir: dir}
}

func (c *fileCoordinator) Acquire(ctx context.Context, key string, ttl time.Duration) (Lease, error) {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir coordinator dir: %w", err)
	}
	path := filepath.Join(c.dir, key+".lock")

	for {
		now := time.Now()
		expires := now.Add(ttl)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = f.WriteString(expires.Format(time.RFC3339Nano))
			_ = f.Close()
			return &fileLease{path: path}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("acquire file lease: %w", err)
		}

		if expired(path, now) {
			_ = os.Remove(path)
			continue
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("acquire file lease: %w", ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func (l *fileLease) Release(_ context.Context) error {
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("release file lease: %w", err)
	}
	return nil
}

func expired(path string, now time.Time) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	t, err := time.Parse(time.RFC3339Nano, string(b))
	if err != nil {
		return true
	}
	return now.After(t)
}

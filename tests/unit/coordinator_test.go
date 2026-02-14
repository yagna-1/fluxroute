package unit

import (
	"context"
	"testing"
	"time"

	"github.com/your-org/agent-router/internal/coordinator"
)

func TestMemoryCoordinatorAcquireRelease(t *testing.T) {
	c := coordinator.NewMemoryCoordinator()
	lease, err := c.Acquire(context.Background(), "k1", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatalf("release lease: %v", err)
	}
}

func TestFileCoordinatorAcquireRelease(t *testing.T) {
	c := coordinator.NewFileCoordinator(t.TempDir())
	lease, err := c.Acquire(context.Background(), "k2", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire file lease: %v", err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatalf("release file lease: %v", err)
	}
}

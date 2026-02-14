package unit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/your-org/fluxroute/internal/coordinator"
)

func TestRedisCoordinatorAcquireRelease(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	c, err := coordinator.NewRedisCoordinator("redis://"+mr.Addr(), "test")
	if err != nil {
		t.Fatalf("new redis coordinator: %v", err)
	}

	lease, err := c.Acquire(context.Background(), "k1", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatalf("release lease: %v", err)
	}
}

func TestRedisCoordinatorMutualExclusion(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	c, err := coordinator.NewRedisCoordinator("redis://"+mr.Addr(), "test")
	if err != nil {
		t.Fatalf("new redis coordinator: %v", err)
	}

	lease, err := c.Acquire(context.Background(), "k2", 2*time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer func() { _ = lease.Release(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if _, err := c.Acquire(ctx, "k2", 2*time.Second); err == nil {
		t.Fatal("expected second acquire to fail under lock contention")
	}
}

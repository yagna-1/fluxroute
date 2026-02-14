package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/your-org/agent-router/internal/controlplane"
)

func main() {
	addr := os.Getenv("CONTROLPLANE_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	svc := controlplane.NewService()
	if err := controlplane.StartServer(ctx, addr, svc); err != nil && err.Error() != "http: Server closed" {
		fmt.Fprintf(os.Stderr, "controlplane failed: %v\n", err)
		os.Exit(1)
	}
}

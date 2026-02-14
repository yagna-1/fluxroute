package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/your-org/agent-router/internal/app"
	"github.com/your-org/agent-router/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Println(version.String())
			return
		case "serve":
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			if err := app.StartRouterServerFromEnv(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintf(os.Stderr, "router server failed: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	manifestPath := "configs/router.example.yaml"
	if len(os.Args) > 1 {
		manifestPath = os.Args[1]
	}
	if v := os.Getenv("MANIFEST_PATH"); v != "" {
		manifestPath = v
	}

	if err := app.RunManifest(manifestPath, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "router failed: %v\n", err)
		os.Exit(1)
	}
}

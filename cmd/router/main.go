package main

import (
	"fmt"
	"os"

	"github.com/your-org/agent-router/internal/app"
)

func main() {
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

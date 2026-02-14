package main

import (
	"fmt"
	"os"

	"github.com/your-org/agent-router/internal/app"
	"github.com/your-org/agent-router/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println(version.String())
		return
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

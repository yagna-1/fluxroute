package main

import (
	"fmt"
	"os"

	"github.com/your-org/agent-router/internal/app"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]
	manifestPath := "configs/router.example.yaml"
	if len(os.Args) > 2 {
		manifestPath = os.Args[2]
	}

	switch command {
	case "run":
		if err := app.RunManifest(manifestPath, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "cli run failed: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := app.ValidateManifest(manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "cli validate failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("manifest is valid: %s\n", manifestPath)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage: agent-router-cli <run|validate> [manifest-path]")
}

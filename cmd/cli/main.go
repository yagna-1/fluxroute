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
	path := "configs/router.example.yaml"
	if len(os.Args) > 2 {
		path = os.Args[2]
	}

	switch command {
	case "run":
		if err := app.RunManifest(path, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "cli run failed: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := app.ValidateManifest(path); err != nil {
			fmt.Fprintf(os.Stderr, "cli validate failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("manifest is valid: %s\n", path)
	case "replay":
		if err := app.ReplayTrace(path, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "cli replay failed: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage: agent-router-cli <run|validate|replay> [manifest-or-trace-path]")
}

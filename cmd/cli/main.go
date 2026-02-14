package main

import (
	"fmt"
	"os"

	"github.com/your-org/agent-router/internal/app"
	"github.com/your-org/agent-router/internal/audit"
	"github.com/your-org/agent-router/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]
	if command == "-v" || command == "--version" || command == "version" {
		fmt.Println(version.String())
		return
	}
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
	case "audit-export":
		inputPath := path
		outputPath := "audit.csv"
		if len(os.Args) > 3 {
			outputPath = os.Args[3]
		}
		if err := audit.ExportJSONLToCSV(inputPath, outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "cli audit-export failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("audit export complete: %s -> %s\n", inputPath, outputPath)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage: agent-router-cli <run|validate|replay|audit-export|version> [path] [output_csv]")
}

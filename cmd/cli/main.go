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
	case "scaffold":
		targetDir := path
		pipelineName := "sample"
		if len(os.Args) > 3 {
			pipelineName = os.Args[3]
		}
		if err := app.ScaffoldProject(targetDir, pipelineName, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "cli scaffold failed: %v\n", err)
			os.Exit(1)
		}
	case "debug":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: agent-router-cli debug <expected_trace> <actual_trace>")
			os.Exit(1)
		}
		if err := app.DebugTrace(os.Args[2], os.Args[3], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "cli debug failed: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage: agent-router-cli <run|validate|replay|audit-export|scaffold|debug|version> [path] [extra]")
}

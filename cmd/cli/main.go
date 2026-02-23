package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/your-org/fluxroute/internal/app"
	"github.com/your-org/fluxroute/internal/audit"
	"github.com/your-org/fluxroute/internal/version"
)

type cliResult struct {
	Command string `json:"command"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	jsonOut := false
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "--json", "-j":
			jsonOut = true
			args = args[1:]
		case "--help", "-h":
			usage(stdout)
			return 0
		case "--version", "-v":
			_, _ = fmt.Fprintln(stdout, version.String())
			return 0
		default:
			return fail(stderr, jsonOut, args[0], "", fmt.Errorf("unknown flag: %s", args[0]))
		}
	}

	if len(args) == 0 {
		usage(stderr)
		return 1
	}

	command := args[0]
	rest := args[1:]
	if command == "version" {
		_, _ = fmt.Fprintln(stdout, version.String())
		return 0
	}

	switch command {
	case "run":
		path := pick(rest, "configs/router.example.yaml", 0)
		if jsonOut {
			report, err := app.RunManifestReport(path)
			if err != nil {
				return fail(stderr, jsonOut, command, path, err)
			}
			failed := 0
			for _, r := range report.Results {
				if r.Err != nil {
					failed++
				}
			}
			return ok(stdout, command, jsonOut, "run completed", map[string]any{
				"manifest_path": path,
				"namespace":     report.Namespace,
				"invocations":   len(report.Results),
				"failed":        failed,
				"metrics":       report.Metrics,
			})
		}
		if err := app.RunManifest(path, stdout); err != nil {
			return fail(stderr, jsonOut, command, path, err)
		}
		return 0
	case "validate":
		path := pick(rest, "configs/router.example.yaml", 0)
		if err := app.ValidateManifest(path); err != nil {
			return fail(stderr, jsonOut, command, path, err)
		}
		if jsonOut {
			return ok(stdout, command, jsonOut, "manifest is valid", map[string]any{"manifest_path": path, "valid": true})
		}
		_, _ = fmt.Fprintf(stdout, "manifest is valid: %s\n", path)
		return 0
	case "replay":
		path := pick(rest, "trace.json", 0)
		if jsonOut {
			var buf bytes.Buffer
			if err := app.ReplayTrace(path, &buf); err != nil {
				return fail(stderr, jsonOut, command, path, err)
			}
			return ok(stdout, command, jsonOut, strings.TrimSpace(buf.String()), map[string]any{"trace_path": path})
		}
		if err := app.ReplayTrace(path, stdout); err != nil {
			return fail(stderr, jsonOut, command, path, err)
		}
		return 0
	case "audit-export":
		inputPath := pick(rest, "audit.log", 0)
		outputPath := pick(rest, "audit.csv", 1)
		if err := audit.ExportJSONLToCSV(inputPath, outputPath); err != nil {
			return fail(stderr, jsonOut, command, inputPath, err)
		}
		if jsonOut {
			return ok(stdout, command, jsonOut, "audit export complete", map[string]any{"input_path": inputPath, "output_path": outputPath})
		}
		_, _ = fmt.Fprintf(stdout, "audit export complete: %s -> %s\n", inputPath, outputPath)
		return 0
	case "scaffold":
		targetDir := pick(rest, "./scaffold-output", 0)
		pipelineName := pick(rest, "sample", 1)
		if err := app.ScaffoldProject(targetDir, pipelineName, stdout); err != nil {
			return fail(stderr, jsonOut, command, targetDir, err)
		}
		if jsonOut {
			return ok(stdout, command, jsonOut, "scaffold generated", map[string]any{"target_dir": targetDir, "pipeline_name": pipelineName})
		}
		return 0
	case "debug":
		if len(rest) < 2 {
			return fail(stderr, jsonOut, command, "", fmt.Errorf("usage: fluxroute-cli debug <expected_trace> <actual_trace>"))
		}
		if jsonOut {
			var buf bytes.Buffer
			if err := app.DebugTrace(rest[0], rest[1], &buf); err != nil {
				return fail(stderr, jsonOut, command, rest[0], err)
			}
			return ok(stdout, command, jsonOut, strings.TrimSpace(buf.String()), map[string]any{"expected_trace": rest[0], "actual_trace": rest[1]})
		}
		if err := app.DebugTrace(rest[0], rest[1], stdout); err != nil {
			return fail(stderr, jsonOut, command, rest[0], err)
		}
		return 0
	default:
		return fail(stderr, jsonOut, command, "", fmt.Errorf("unknown command: %s", command))
	}
}

func pick(args []string, fallback string, idx int) string {
	if len(args) > idx && strings.TrimSpace(args[idx]) != "" {
		return args[idx]
	}
	return fallback
}

func ok(out io.Writer, command string, jsonOut bool, message string, data any) int {
	if !jsonOut {
		if message != "" {
			_, _ = fmt.Fprintln(out, message)
		}
		return 0
	}
	_ = json.NewEncoder(out).Encode(cliResult{Command: command, Status: "ok", Message: message, Data: data})
	return 0
}

func fail(out io.Writer, jsonOut bool, command string, path string, err error) int {
	if jsonOut {
		_ = json.NewEncoder(out).Encode(cliResult{Command: command, Status: "error", Error: err.Error(), Data: map[string]any{"path": path}})
		return 1
	}
	if path == "" {
		_, _ = fmt.Fprintf(out, "cli %s failed: %v\n", command, err)
	} else {
		_, _ = fmt.Fprintf(out, "cli %s failed for %s: %v\n", command, path, err)
	}
	_, _ = fmt.Fprintln(out, "use `fluxroute-cli --help` for command examples")
	return 1
}

func usage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "FluxRoute CLI")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Usage:")
	_, _ = fmt.Fprintln(out, "  fluxroute-cli [--json] <command> [args]")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Commands:")
	_, _ = fmt.Fprintln(out, "  run [manifest_path]                    Execute a manifest")
	_, _ = fmt.Fprintln(out, "  validate [manifest_path]               Validate manifest only")
	_, _ = fmt.Fprintln(out, "  replay [trace_path]                    Replay a trace and verify outputs")
	_, _ = fmt.Fprintln(out, "  audit-export [jsonl_path] [csv_path]   Export audit JSONL to CSV")
	_, _ = fmt.Fprintln(out, "  scaffold [target_dir] [pipeline_name]  Generate a starter pipeline")
	_, _ = fmt.Fprintln(out, "  debug <expected_trace> <actual_trace>  Show replay divergence")
	_, _ = fmt.Fprintln(out, "  version                                Print CLI version")
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, "Examples:")
	_, _ = fmt.Fprintln(out, "  fluxroute-cli run configs/router.example.yaml")
	_, _ = fmt.Fprintln(out, "  fluxroute-cli --json validate configs/router.example.yaml")
	_, _ = fmt.Fprintln(out, "  fluxroute-cli replay trace.json")
	_, _ = fmt.Fprintln(out, "  fluxroute-cli audit-export audit.log audit.csv")
	_, _ = fmt.Fprintln(out, "  fluxroute-cli scaffold ./generated customer-support")
	_, _ = fmt.Fprintln(out, "  fluxroute-cli debug expected.json actual.json")
}

package app

import (
	"fmt"
	"io"

	"github.com/your-org/fluxroute/internal/scaffold"
	"github.com/your-org/fluxroute/internal/trace"
)

func ScaffoldProject(targetDir string, pipelineName string, out io.Writer) error {
	if err := scaffold.Generate(targetDir, pipelineName); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "scaffold generated at %s\n", targetDir)
	return nil
}

func DebugTrace(expectedPath string, actualPath string, out io.Writer) error {
	expected, err := trace.LoadFromFile(expectedPath)
	if err != nil {
		return err
	}
	actual, err := trace.LoadFromFile(actualPath)
	if err != nil {
		return err
	}
	div := trace.Compare(expected, actual)
	_, _ = fmt.Fprintln(out, trace.FormatDivergence(div))
	if len(div) > 0 {
		return fmt.Errorf("trace divergence found: %d issue(s)", len(div))
	}
	return nil
}

package audit

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
)

// ExportJSONLToCSV converts line-delimited JSON audit logs into CSV.
func ExportJSONLToCSV(inputPath string, outputPath string) error {
	in, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input audit log: %w", err)
	}
	defer in.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output csv: %w", err)
	}
	defer out.Close()

	w := csv.NewWriter(out)
	defer w.Flush()
	if err := w.Write([]string{"ts", "actor", "action", "resource", "status", "error"}); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	s := bufio.NewScanner(in)
	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return fmt.Errorf("parse audit line: %w", err)
		}
		if err := w.Write([]string{ev.Timestamp, ev.Actor, ev.Action, ev.Resource, ev.Status, ev.Error}); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("scan audit log: %w", err)
	}
	return nil
}

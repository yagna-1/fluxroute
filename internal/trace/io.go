package trace

import (
	"encoding/json"
	"fmt"
	"os"
)

func SaveToFile(path string, tr ExecutionTrace) error {
	b, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return fmt.Errorf("trace: marshal: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("trace: write %q: %w", path, err)
	}
	return nil
}

func LoadFromFile(path string) (ExecutionTrace, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ExecutionTrace{}, fmt.Errorf("trace: read %q: %w", path, err)
	}
	var tr ExecutionTrace
	if err := json.Unmarshal(b, &tr); err != nil {
		return ExecutionTrace{}, fmt.Errorf("trace: unmarshal %q: %w", path, err)
	}
	return tr, nil
}

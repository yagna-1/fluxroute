package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AstraGraphAuditEntry struct {
	WorkflowID     string          `json:"workflow_id"`
	AgentID        string          `json:"agent_id"`
	ToolName       string          `json:"tool_name"`
	Arguments      json.RawMessage `json:"arguments"`
	Status         string          `json:"status"` // "allowed" | "blocked"
	DeviationScore float32         `json:"deviation_score"`
	Timestamp      string          `json:"timestamp"`
}

// ExportAstraGraphAudit writes FluxRoute execution trace as AstraGraph audit JSON and optionally posts it.
func ExportAstraGraphAudit(execTrace ExecutionTrace, namespace string) (string, error) {
	entries := buildAstraGraphEntries(execTrace, namespace)
	if len(entries) == 0 {
		return "", nil
	}

	auditPath := strings.TrimSpace(os.Getenv("ASTRAGRAPH_AUDIT_PATH"))
	var resolvedPath string
	if auditPath != "" {
		resolvedPath = strings.ReplaceAll(auditPath, "{task_id}", safeFilePart(execTrace.TaskID))
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Clean(resolvedPath)
		}
		if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
			return "", fmt.Errorf("astragraph export mkdir: %w", err)
		}
		payload, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return "", fmt.Errorf("astragraph export marshal: %w", err)
		}
		if err := os.WriteFile(resolvedPath, payload, 0o644); err != nil {
			return "", fmt.Errorf("astragraph export write: %w", err)
		}
	}

	endpoint := strings.TrimSpace(os.Getenv("ASTRAGRAPH_AUDIT_ENDPOINT"))
	if endpoint != "" {
		payload, err := json.Marshal(entries)
		if err != nil {
			return resolvedPath, fmt.Errorf("astragraph export marshal post payload: %w", err)
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			return resolvedPath, fmt.Errorf("astragraph export request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if token := strings.TrimSpace(os.Getenv("ASTRAGRAPH_TOKEN")); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return resolvedPath, fmt.Errorf("astragraph export post: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return resolvedPath, fmt.Errorf("astragraph export post failed: status=%d", resp.StatusCode)
		}
	}

	return resolvedPath, nil
}

func buildAstraGraphEntries(execTrace ExecutionTrace, namespace string) []AstraGraphAuditEntry {
	entries := make([]AstraGraphAuditEntry, 0, len(execTrace.Steps))
	for _, step := range execTrace.Steps {
		toolName := step.Input.Metadata["tool_name"]
		if toolName == "" {
			toolName = step.AgentID
		}
		if toolName == "" {
			toolName = "unknown_tool"
		}

		arguments := normalizePayload(step.Input.Payload)
		if step.Input.Metadata != nil && namespace != "" {
			var parsed map[string]any
			if err := json.Unmarshal(arguments, &parsed); err == nil {
				if _, exists := parsed["namespace"]; !exists {
					parsed["namespace"] = namespace
				}
				if b, err := json.Marshal(parsed); err == nil {
					arguments = b
				}
			}
		}

		status := "allowed"
		score := float32(0.0)
		if strings.TrimSpace(step.Error) != "" {
			status = "blocked"
			score = 1.0
		}

		ts := step.Input.Timestamp.UTC().Format(time.RFC3339Nano)
		if ts == "0001-01-01T00:00:00Z" {
			ts = time.Now().UTC().Format(time.RFC3339Nano)
		}

		entries = append(entries, AstraGraphAuditEntry{
			WorkflowID:     execTrace.TaskID,
			AgentID:        step.AgentID,
			ToolName:       toolName,
			Arguments:      json.RawMessage(arguments),
			Status:         status,
			DeviationScore: score,
			Timestamp:      ts,
		})
	}
	return entries
}

func normalizePayload(payload []byte) []byte {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return []byte("{}")
	}
	var anyJSON any
	if err := json.Unmarshal(trimmed, &anyJSON); err == nil {
		if obj, ok := anyJSON.(map[string]any); ok {
			b, _ := json.Marshal(obj)
			return b
		}
	}
	b, _ := json.Marshal(map[string]any{"payload": string(trimmed)})
	return b
}

func safeFilePart(in string) string {
	if in == "" {
		return "trace"
	}
	in = strings.ToLower(in)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteRune('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "trace"
	}
	return out
}

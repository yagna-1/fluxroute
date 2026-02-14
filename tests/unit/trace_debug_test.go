package unit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/your-org/agent-router/internal/app"
	"github.com/your-org/agent-router/internal/trace"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

func TestDebugTraceDetectsDivergence(t *testing.T) {
	dir := t.TempDir()
	expPath := filepath.Join(dir, "exp.json")
	actPath := filepath.Join(dir, "act.json")

	exp := trace.ExecutionTrace{TaskID: "t1", StartTime: time.Now(), EndTime: time.Now(), Steps: []trace.Step{{InvocationID: "1", AgentID: "a", Output: agentfunc.AgentOutput{RequestID: "r1", Payload: []byte("ok")}, Attempt: 1}}}
	act := trace.ExecutionTrace{TaskID: "t1", StartTime: time.Now(), EndTime: time.Now(), Steps: []trace.Step{{InvocationID: "1", AgentID: "a", Output: agentfunc.AgentOutput{RequestID: "r1", Payload: []byte("changed")}, Attempt: 1}}}

	if err := trace.SaveToFile(expPath, exp); err != nil {
		t.Fatalf("save exp trace: %v", err)
	}
	if err := trace.SaveToFile(actPath, act); err != nil {
		t.Fatalf("save act trace: %v", err)
	}

	var out bytes.Buffer
	err := app.DebugTrace(expPath, actPath, &out)
	if err == nil {
		t.Fatal("expected divergence error")
	}
	if !bytes.Contains(out.Bytes(), []byte("payload_hash")) {
		t.Fatalf("expected payload_hash divergence in output: %s", out.String())
	}
}

func TestDebugTraceNoDivergence(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.json")
	p2 := filepath.Join(dir, "b.json")

	tr := trace.ExecutionTrace{TaskID: "t1", StartTime: time.Now(), EndTime: time.Now(), Steps: []trace.Step{{InvocationID: "1", AgentID: "a", Output: agentfunc.AgentOutput{RequestID: "r1", Payload: []byte("ok")}, Attempt: 1}}}
	if err := trace.SaveToFile(p1, tr); err != nil {
		t.Fatalf("save trace 1: %v", err)
	}
	if err := trace.SaveToFile(p2, tr); err != nil {
		t.Fatalf("save trace 2: %v", err)
	}

	var out bytes.Buffer
	if err := app.DebugTrace(p1, p2, &out); err != nil {
		t.Fatalf("expected no divergence: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("no divergence")) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestDebugTraceMissingFile(t *testing.T) {
	var out bytes.Buffer
	if err := app.DebugTrace("/nonexistent/a.json", "/nonexistent/b.json", &out); err == nil {
		t.Fatal("expected error for missing files")
	}
}

func TestDebugTraceCreatesNoFiles(t *testing.T) {
	dir := t.TempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatal("expected empty temp dir")
	}
}

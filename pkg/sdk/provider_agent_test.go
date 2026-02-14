package sdk

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/your-org/agent-router/pkg/adapters"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

type fakeProvider struct{}

func (fakeProvider) Name() string { return "fake" }
func (fakeProvider) Generate(_ context.Context, req adapters.GenerateRequest) (adapters.GenerateResponse, error) {
	return adapters.GenerateResponse{Text: "echo:" + req.Prompt, InputTokens: 1, OutputTokens: 2}, nil
}

func TestAgentFromProvider(t *testing.T) {
	agent := AgentFromProvider(fakeProvider{}, "fake-model")
	out, err := agent(context.Background(), agentfunc.AgentInput{
		RequestID: "req_1",
		Payload:   []byte(`{"prompt":"hello"}`),
	})
	if err != nil {
		t.Fatalf("agent failed: %v", err)
	}

	var payload CompletionPayload
	if err := json.Unmarshal(out.Payload, &payload); err != nil {
		t.Fatalf("unmarshal output payload: %v", err)
	}
	if payload.Text != "echo:hello" || payload.Provider != "fake" || payload.Model != "fake-model" {
		t.Fatalf("unexpected output payload: %+v", payload)
	}
}

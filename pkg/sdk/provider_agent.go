package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/your-org/agent-router/pkg/adapters"
	"github.com/your-org/agent-router/pkg/agentfunc"
)

// PromptPayload is the expected payload shape for adapter-backed agents.
type PromptPayload struct {
	Prompt string `json:"prompt"`
}

// CompletionPayload is the normalized output payload shape.
type CompletionPayload struct {
	Text         string `json:"text"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
}

// AgentFromProvider converts a provider adapter to a runtime agent function.
func AgentFromProvider(provider adapters.Provider, model string) agentfunc.AgentFunc {
	return func(ctx context.Context, input agentfunc.AgentInput) (agentfunc.AgentOutput, error) {
		if provider == nil {
			return agentfunc.AgentOutput{}, fmt.Errorf("provider is nil")
		}
		select {
		case <-ctx.Done():
			return agentfunc.AgentOutput{}, ctx.Err()
		default:
		}

		prompt, err := extractPrompt(input.Payload)
		if err != nil {
			return agentfunc.AgentOutput{}, err
		}
		resp, err := provider.Generate(ctx, adapters.GenerateRequest{Model: model, Prompt: prompt})
		if err != nil {
			return agentfunc.AgentOutput{}, err
		}

		payload, err := json.Marshal(CompletionPayload{
			Text:         resp.Text,
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
			Provider:     provider.Name(),
			Model:        model,
		})
		if err != nil {
			return agentfunc.AgentOutput{}, fmt.Errorf("marshal completion payload: %w", err)
		}
		return agentfunc.AgentOutput{RequestID: input.RequestID, Payload: payload}, nil
	}
}

func extractPrompt(payload []byte) (string, error) {
	if len(payload) == 0 {
		return "", fmt.Errorf("empty payload")
	}

	var p PromptPayload
	if err := json.Unmarshal(payload, &p); err == nil && strings.TrimSpace(p.Prompt) != "" {
		return p.Prompt, nil
	}

	trimmed := strings.TrimSpace(string(payload))
	if trimmed != "" {
		return trimmed, nil
	}
	return "", fmt.Errorf("prompt not found in payload")
}

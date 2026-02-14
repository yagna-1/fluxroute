package adapters

import "context"

// GenerateRequest is a provider-agnostic text generation request.
type GenerateRequest struct {
	Model       string
	Prompt      string
	MaxTokens   int
	Temperature float64
}

// GenerateResponse is a provider-agnostic generation response.
type GenerateResponse struct {
	Text         string
	InputTokens  int
	OutputTokens int
	Raw          []byte
}

// Provider is the common interface all LLM adapters must satisfy.
type Provider interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
}

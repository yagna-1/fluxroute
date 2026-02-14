package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/your-org/agent-router/pkg/adapters"
)

const defaultBaseURL = "https://api.anthropic.com"

// Client implements adapters.Provider for Anthropic Messages API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string, httpClient *http.Client, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{apiKey: apiKey, httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *Client) Name() string { return "anthropic" }

func (c *Client) Generate(ctx context.Context, req adapters.GenerateRequest) (adapters.GenerateResponse, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return adapters.GenerateResponse{}, adapters.ErrMissingAPIKey
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return adapters.GenerateResponse{}, adapters.ErrEmptyPrompt
	}
	if req.Model == "" {
		req.Model = "claude-3-5-sonnet-latest"
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 512
	}

	url := c.baseURL + "/v1/messages"
	hReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return adapters.GenerateResponse{}, fmt.Errorf("build request: %w", err)
	}
	hReq.Header.Set("x-api-key", c.apiKey)
	hReq.Header.Set("anthropic-version", "2023-06-01")

	payload := map[string]any{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"messages": []map[string]any{{
			"role":    "user",
			"content": req.Prompt,
		}},
	}
	body, err := adapters.DoJSON(ctx, c.httpClient, hReq, payload)
	if err != nil {
		return adapters.GenerateResponse{}, err
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return adapters.GenerateResponse{}, fmt.Errorf("parse response: %w", err)
	}

	text := ""
	for _, c := range parsed.Content {
		if c.Type == "text" || c.Text != "" {
			text += c.Text
		}
	}

	return adapters.GenerateResponse{
		Text:         text,
		InputTokens:  parsed.Usage.InputTokens,
		OutputTokens: parsed.Usage.OutputTokens,
		Raw:          body,
	}, nil
}

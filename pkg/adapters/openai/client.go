package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/your-org/agent-router/pkg/adapters"
)

const defaultBaseURL = "https://api.openai.com"

// Client implements adapters.Provider for OpenAI Responses API.
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

func (c *Client) Name() string { return "openai" }

func (c *Client) Generate(ctx context.Context, req adapters.GenerateRequest) (adapters.GenerateResponse, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return adapters.GenerateResponse{}, adapters.ErrMissingAPIKey
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return adapters.GenerateResponse{}, adapters.ErrEmptyPrompt
	}
	if req.Model == "" {
		req.Model = "gpt-4o-mini"
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 512
	}

	url := c.baseURL + "/v1/responses"
	hReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return adapters.GenerateResponse{}, fmt.Errorf("build request: %w", err)
	}
	hReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	payload := map[string]any{
		"model":             req.Model,
		"input":             req.Prompt,
		"max_output_tokens": req.MaxTokens,
		"temperature":       req.Temperature,
	}
	body, err := adapters.DoJSON(ctx, c.httpClient, hReq, payload)
	if err != nil {
		return adapters.GenerateResponse{}, err
	}

	var parsed struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		OutputText string `json:"output_text"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return adapters.GenerateResponse{}, fmt.Errorf("parse response: %w", err)
	}

	text := strings.TrimSpace(parsed.OutputText)
	if text == "" {
		for _, o := range parsed.Output {
			for _, c := range o.Content {
				if c.Type == "output_text" || c.Text != "" {
					text += c.Text
				}
			}
		}
	}

	return adapters.GenerateResponse{
		Text:         text,
		InputTokens:  parsed.Usage.InputTokens,
		OutputTokens: parsed.Usage.OutputTokens,
		Raw:          body,
	}, nil
}

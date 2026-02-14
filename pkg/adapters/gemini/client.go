package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/your-org/agent-router/pkg/adapters"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

// Client implements adapters.Provider for Gemini generateContent API.
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

func (c *Client) Name() string { return "gemini" }

func (c *Client) Generate(ctx context.Context, req adapters.GenerateRequest) (adapters.GenerateResponse, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return adapters.GenerateResponse{}, adapters.ErrMissingAPIKey
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return adapters.GenerateResponse{}, adapters.ErrEmptyPrompt
	}
	if req.Model == "" {
		req.Model = "gemini-1.5-pro"
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 512
	}

	urlStr := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", c.baseURL, url.PathEscape(req.Model), url.QueryEscape(c.apiKey))
	hReq, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, nil)
	if err != nil {
		return adapters.GenerateResponse{}, fmt.Errorf("build request: %w", err)
	}

	payload := map[string]any{
		"contents": []map[string]any{{
			"role":  "user",
			"parts": []map[string]any{{"text": req.Prompt}},
		}},
		"generationConfig": map[string]any{
			"temperature":     req.Temperature,
			"maxOutputTokens": req.MaxTokens,
		},
	}
	body, err := adapters.DoJSON(ctx, c.httpClient, hReq, payload)
	if err != nil {
		return adapters.GenerateResponse{}, err
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return adapters.GenerateResponse{}, fmt.Errorf("parse response: %w", err)
	}

	text := ""
	for _, cand := range parsed.Candidates {
		for _, p := range cand.Content.Parts {
			text += p.Text
		}
	}

	return adapters.GenerateResponse{
		Text:         text,
		InputTokens:  parsed.UsageMetadata.PromptTokenCount,
		OutputTokens: parsed.UsageMetadata.CandidatesTokenCount,
		Raw:          body,
	}, nil
}

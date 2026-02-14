package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DoJSON sends JSON request payload and returns response body.
func DoJSON(ctx context.Context, client *http.Client, req *http.Request, payload any) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}

	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(b))
		req.ContentLength = int64(len(b))
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

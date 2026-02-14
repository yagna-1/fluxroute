package gemini

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/your-org/agent-router/pkg/adapters"
)

func TestGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.String(), "key=test-key") {
			t.Fatalf("missing API key query param: %s", r.URL.String())
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "hello") {
			t.Fatalf("request body missing prompt: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"world"}]}}],"usageMetadata":{"promptTokenCount":7,"candidatesTokenCount":8}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.Client(), srv.URL)
	resp, err := c.Generate(context.Background(), adapters.GenerateRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.Text != "world" || resp.InputTokens != 7 || resp.OutputTokens != 8 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

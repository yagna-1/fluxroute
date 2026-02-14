package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/your-org/fluxroute/pkg/adapters"
)

func TestGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("unexpected api key header: %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "hello") {
			t.Fatalf("request body missing prompt: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"world"}],"usage":{"input_tokens":2,"output_tokens":5}}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.Client(), srv.URL)
	resp, err := c.Generate(context.Background(), adapters.GenerateRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp.Text != "world" || resp.InputTokens != 2 || resp.OutputTokens != 5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

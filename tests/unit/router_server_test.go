package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/fluxroute/internal/app"
)

func TestRouterHandlerHealthAndReady(t *testing.T) {
	h := app.RouterHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for healthz, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for readyz, got %d", w.Code)
	}
}

func TestRouterHandlerValidateRejectsMissingBody(t *testing.T) {
	h := app.RouterHandler()
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader([]byte("{")))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", w.Code)
	}
}

func TestRouterHandlerReplayRequiresTracePath(t *testing.T) {
	h := app.RouterHandler()
	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/replay", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing trace_path, got %d", w.Code)
	}
}

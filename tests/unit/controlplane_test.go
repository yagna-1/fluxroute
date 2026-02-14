package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/your-org/fluxroute/internal/controlplane"
)

func TestControlplaneHandler(t *testing.T) {
	svc := controlplane.NewService()
	h := svc.Handler()

	postBody, _ := json.Marshal(map[string]any{"id": "tenant-a"})
	req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(postBody))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	usageBody, _ := json.Marshal(map[string]any{"tenant_id": "tenant-a", "invocations": 5})
	req = httptest.NewRequest(http.MethodPost, "/usage", bytes.NewReader(usageBody))
	req.Header.Set("X-Role", "admin")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/usage?tenant_id=tenant-a", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("5")) {
		t.Fatalf("expected usage response to include 5, got %s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/billing/rates", bytes.NewReader([]byte(`{"usd_per_thousand":2.5}`)))
	req.Header.Set("X-Role", "admin")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for billing rate update, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/billing/invoice?tenant_id=tenant-a", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for invoice, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("amount_usd")) {
		t.Fatalf("expected invoice payload, got %s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for healthz, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/sla", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for sla, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("slo_target")) {
		t.Fatalf("expected sla payload, got %s", w.Body.String())
	}
}

func TestControlplaneRBACDenied(t *testing.T) {
	svc := controlplane.NewService()
	h := svc.Handler()

	postBody, _ := json.Marshal(map[string]any{"id": "tenant-b"})
	req := httptest.NewRequest(http.MethodPost, "/tenants", bytes.NewReader(postBody))
	req.Header.Set("X-Role", "operator")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

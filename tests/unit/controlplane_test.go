package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/your-org/fluxroute/internal/controlplane"
)

func TestControlplaneHandler(t *testing.T) {
	t.Setenv("CONTROLPLANE_API_KEY", "")
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

	req = httptest.NewRequest(http.MethodGet, "/v1/tenants", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /v1/tenants, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/usage?page=1&page_size=10", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for paginated usage listing, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/billing/invoice?tenant_id=tenant-a&format=csv", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for invoice csv, got %d", w.Code)
	}
	if ctype := w.Header().Get("Content-Type"); ctype != "text/csv" {
		t.Fatalf("expected text/csv content type, got %q", ctype)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/billing/summary", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /v1/billing/summary, got %d", w.Code)
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

func TestControlplaneAPIKeyMiddleware(t *testing.T) {
	t.Setenv("CONTROLPLANE_API_KEY", "top-secret")
	svc := controlplane.NewService()
	h := svc.Handler()

	req := httptest.NewRequest(http.MethodGet, "/tenants", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without api key, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/tenants", nil)
	req.Header.Set("X-API-Key", "top-secret")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid api key, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/tenants", nil)
	req.Header.Set("Authorization", "Bearer top-secret")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with bearer api key, got %d", w.Code)
	}
}

func TestControlplaneBillingSummary(t *testing.T) {
	t.Setenv("CONTROLPLANE_API_KEY", "")
	svc := controlplane.NewService()
	if err := svc.AddTenant("tenant-s"); err != nil {
		t.Fatalf("add tenant: %v", err)
	}
	now := time.Now().UTC()
	month := now.Format("2006-01")
	if err := svc.AddUsageAt("tenant-s", 7, now); err != nil {
		t.Fatalf("add usage at: %v", err)
	}

	h := svc.Handler()
	req := httptest.NewRequest(http.MethodGet, "/billing/summary?month="+month, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for billing summary, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("\"grand_total_invocations\":7")) {
		t.Fatalf("expected summary total in response, got %s", w.Body.String())
	}
}

func TestControlplaneRejectsNonPositiveUsage(t *testing.T) {
	t.Setenv("CONTROLPLANE_API_KEY", "")
	svc := controlplane.NewService()
	if err := svc.AddTenant("tenant-x"); err != nil {
		t.Fatalf("add tenant: %v", err)
	}

	h := svc.Handler()
	usageBody, _ := json.Marshal(map[string]any{"tenant_id": "tenant-x", "invocations": 0})
	req := httptest.NewRequest(http.MethodPost, "/usage", bytes.NewReader(usageBody))
	req.Header.Set("X-Role", "admin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-positive usage, got %d", w.Code)
	}
}

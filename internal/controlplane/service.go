package controlplane

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/your-org/fluxroute/internal/billing"
	"github.com/your-org/fluxroute/internal/security"
)

type usageEvent struct {
	TenantID    string
	Invocations int64
	OccurredAt  time.Time
}

type usageRow struct {
	TenantID    string `json:"tenant_id"`
	Invocations int64  `json:"invocations"`
}

type billSummaryRow struct {
	TenantID    string `json:"tenant_id"`
	Invocations int64  `json:"invocations"`
}

type Service struct {
	mu         sync.Mutex
	tenants    map[string]struct{}
	usage      map[string]int64
	usageEvent []usageEvent
	rate       billing.RateCard
	started    time.Time
	reqs       int64
}

func NewService() *Service {
	rate, _ := billing.NewRateCard(1.0)
	return &Service{tenants: make(map[string]struct{}), usage: make(map[string]int64), rate: rate, started: time.Now()}
}

func (s *Service) AddTenant(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		return fmt.Errorf("tenant id is empty")
	}
	if _, exists := s.tenants[id]; exists {
		return fmt.Errorf("tenant %q already exists", id)
	}
	s.tenants[id] = struct{}{}
	return nil
}

func (s *Service) ListTenants() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.tenants))
	for id := range s.tenants {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *Service) AddUsage(tenantID string, invocations int64) error {
	return s.AddUsageAt(tenantID, invocations, time.Now().UTC())
}

func (s *Service) AddUsageAt(tenantID string, invocations int64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tenantID == "" {
		return fmt.Errorf("tenant id is empty")
	}
	if invocations <= 0 {
		return fmt.Errorf("invocations must be > 0")
	}
	if _, ok := s.tenants[tenantID]; !ok {
		return fmt.Errorf("tenant %q not found", tenantID)
	}
	s.usage[tenantID] += invocations
	s.usageEvent = append(s.usageEvent, usageEvent{TenantID: tenantID, Invocations: invocations, OccurredAt: at.UTC()})
	return nil
}

func (s *Service) Usage(tenantID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage[tenantID]
}

func (s *Service) UsageRows(query string) []usageRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]usageRow, 0, len(s.usage))
	for tenantID, invocations := range s.usage {
		if query != "" && !strings.Contains(strings.ToLower(tenantID), strings.ToLower(query)) {
			continue
		}
		rows = append(rows, usageRow{TenantID: tenantID, Invocations: invocations})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].TenantID < rows[j].TenantID })
	return rows
}

func (s *Service) SetRate(usdPerThousand float64) error {
	rate, err := billing.NewRateCard(usdPerThousand)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rate = rate
	return nil
}

func (s *Service) Invoice(tenantID string) billing.Invoice {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rate.Invoice(tenantID, s.usage[tenantID])
}

func (s *Service) MonthlyUsageRows(month string) ([]billSummaryRow, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	monthStart, err := parseMonthStart(month)
	if err != nil {
		return nil, "", err
	}
	monthEnd := monthStart.AddDate(0, 1, 0)

	totals := map[string]int64{}
	for _, ev := range s.usageEvent {
		if !ev.OccurredAt.Before(monthStart) && ev.OccurredAt.Before(monthEnd) {
			totals[ev.TenantID] += ev.Invocations
		}
	}

	rows := make([]billSummaryRow, 0, len(totals))
	for tenantID, invocations := range totals {
		rows = append(rows, billSummaryRow{TenantID: tenantID, Invocations: invocations})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].TenantID < rows[j].TenantID })
	return rows, monthStart.Format("2006-01"), nil
}

func (s *Service) Handler() http.Handler {
	policy := security.DefaultPolicy()
	apiKey := strings.TrimSpace(os.Getenv("CONTROLPLANE_API_KEY"))

	requireAPIKey := func(w http.ResponseWriter, r *http.Request) bool {
		if apiKey == "" {
			return true
		}
		provided := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if provided == "" {
			authz := strings.TrimSpace(r.Header.Get("Authorization"))
			const prefix = "Bearer "
			if strings.HasPrefix(authz, prefix) {
				provided = strings.TrimSpace(strings.TrimPrefix(authz, prefix))
			}
		}
		if provided != apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		return true
	}

	requireAdmin := func(w http.ResponseWriter, r *http.Request) bool {
		role, err := security.ParseRole(r.Header.Get("X-Role"))
		if err != nil {
			role = security.RoleViewer
		}
		if !policy.IsAllowed(role, security.ActionAdmin) {
			http.Error(w, "rbac denied", http.StatusForbidden)
			return false
		}
		return true
	}

	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}

	register := func(mux *http.ServeMux, path string, h http.HandlerFunc) {
		mux.HandleFunc(path, h)
		mux.HandleFunc("/v1"+path, h)
	}

	mux := http.NewServeMux()
	register(mux, "/healthz", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	register(mux, "/readyz", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	register(mux, "/sla", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		uptime := int64(time.Since(s.started).Seconds())
		writeJSON(w, map[string]any{
			"uptime_seconds": uptime,
			"total_requests": atomic.LoadInt64(&s.reqs),
			"slo_target":     "99.9%",
		})
	})
	register(mux, "/tenants", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		if !requireAPIKey(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			q := strings.TrimSpace(r.URL.Query().Get("q"))
			page, pageSize := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("page_size"))
			tenants := s.ListTenants()
			filtered := make([]string, 0, len(tenants))
			for _, id := range tenants {
				if q != "" && !strings.Contains(strings.ToLower(id), strings.ToLower(q)) {
					continue
				}
				filtered = append(filtered, id)
			}
			total := len(filtered)
			filtered = paginateStrings(filtered, page, pageSize)
			writeJSON(w, map[string]any{"tenants": filtered, "total": total, "page": page, "page_size": pageSize})
		case http.MethodPost:
			if !requireAdmin(w, r) {
				return
			}
			var req struct {
				ID string `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := s.AddTenant(req.ID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	register(mux, "/usage", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		if !requireAPIKey(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
			if tenantID != "" {
				writeJSON(w, map[string]any{"tenant_id": tenantID, "invocations": s.Usage(tenantID)})
				return
			}
			q := strings.TrimSpace(r.URL.Query().Get("q"))
			page, pageSize := parsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("page_size"))
			rows := s.UsageRows(q)
			total := len(rows)
			rows = paginateUsageRows(rows, page, pageSize)
			writeJSON(w, map[string]any{"items": rows, "total": total, "page": page, "page_size": pageSize})
		case http.MethodPost:
			if !requireAdmin(w, r) {
				return
			}
			var req struct {
				TenantID    string `json:"tenant_id"`
				Invocations int64  `json:"invocations"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := s.AddUsage(req.TenantID, req.Invocations); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	register(mux, "/billing/rates", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		if !requireAPIKey(w, r) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.mu.Lock()
			rate := s.rate.USDPerThousand
			s.mu.Unlock()
			writeJSON(w, map[string]any{"usd_per_thousand": rate})
		case http.MethodPost:
			if !requireAdmin(w, r) {
				return
			}
			var req struct {
				USDPerThousand float64 `json:"usd_per_thousand"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := s.SetRate(req.USDPerThousand); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	register(mux, "/billing/invoice", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		if !requireAPIKey(w, r) {
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		invoice := s.Invoice(tenantID)
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "csv") {
			csvBytes, err := invoiceCSV(invoice)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/csv")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(csvBytes)
			return
		}
		writeJSON(w, invoice)
	})
	register(mux, "/billing/summary", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.reqs, 1)
		if !requireAPIKey(w, r) {
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rows, month, err := s.MonthlyUsageRows(strings.TrimSpace(r.URL.Query().Get("month")))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		grand := int64(0)
		for _, row := range rows {
			grand += row.Invocations
		}
		writeJSON(w, map[string]any{
			"month":                   month,
			"totals":                  rows,
			"grand_total_invocations": grand,
		})
	})
	return mux
}

func StartServer(ctx context.Context, addr string, svc *Service) error {
	if addr == "" {
		addr = ":8081"
	}
	s := &http.Server{Addr: addr, Handler: svc.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()
	return s.ListenAndServe()
}

func StartServerTLS(ctx context.Context, addr string, svc *Service, certFile string, keyFile string, caFile string, requireClientCert bool) error {
	if addr == "" {
		addr = ":8081"
	}
	tlsCfg, err := security.BuildServerTLSConfig(certFile, keyFile, caFile, requireClientCert)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen controlplane: %w", err)
	}
	tlsListener := tls.NewListener(ln, tlsCfg)
	s := &http.Server{Addr: ln.Addr().String(), Handler: svc.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()
	return s.Serve(tlsListener)
}

func parsePagination(pageRaw string, sizeRaw string) (int, int) {
	page := 1
	pageSize := 50
	if p, err := strconv.Atoi(strings.TrimSpace(pageRaw)); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(strings.TrimSpace(sizeRaw)); err == nil && s > 0 {
		if s > 200 {
			s = 200
		}
		pageSize = s
	}
	return page, pageSize
}

func paginateStrings(in []string, page int, pageSize int) []string {
	start := (page - 1) * pageSize
	if start >= len(in) {
		return []string{}
	}
	end := start + pageSize
	if end > len(in) {
		end = len(in)
	}
	return in[start:end]
}

func paginateUsageRows(in []usageRow, page int, pageSize int) []usageRow {
	start := (page - 1) * pageSize
	if start >= len(in) {
		return []usageRow{}
	}
	end := start + pageSize
	if end > len(in) {
		end = len(in)
	}
	return in[start:end]
}

func parseMonthStart(month string) (time.Time, error) {
	if month == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	}
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month format, expected YYYY-MM")
	}
	return t.UTC(), nil
}

func invoiceCSV(invoice billing.Invoice) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	w := csv.NewWriter(buf)
	if err := w.Write([]string{"tenant_id", "invocations", "usd_per_thousand", "amount_usd"}); err != nil {
		return nil, err
	}
	if err := w.Write([]string{
		invoice.TenantID,
		strconv.FormatInt(invoice.Invocations, 10),
		fmt.Sprintf("%.4f", invoice.USDPerThousand),
		fmt.Sprintf("%.4f", invoice.AmountUSD),
	}); err != nil {
		return nil, err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

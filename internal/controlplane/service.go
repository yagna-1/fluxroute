package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/your-org/agent-router/internal/security"
)

type Service struct {
	mu      sync.Mutex
	tenants map[string]struct{}
	usage   map[string]int64
}

func NewService() *Service {
	return &Service{tenants: make(map[string]struct{}), usage: make(map[string]int64)}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[tenantID]; !ok {
		return fmt.Errorf("tenant %q not found", tenantID)
	}
	s.usage[tenantID] += invocations
	return nil
}

func (s *Service) Usage(tenantID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage[tenantID]
}

func (s *Service) Handler() http.Handler {
	policy := security.DefaultPolicy()

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

	mux := http.NewServeMux()
	mux.HandleFunc("/tenants", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"tenants": s.ListTenants()})
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

	mux.HandleFunc("/usage", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tenantID := r.URL.Query().Get("tenant_id")
			_ = json.NewEncoder(w).Encode(map[string]any{"tenant_id": tenantID, "invocations": s.Usage(tenantID)})
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
	return mux
}

func StartServer(ctx context.Context, addr string, svc *Service) error {
	if addr == "" {
		addr = ":8081"
	}
	s := &http.Server{Addr: addr, Handler: svc.Handler()}
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()
	return s.ListenAndServe()
}

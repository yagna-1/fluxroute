package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/your-org/agent-router/internal/security"
)

func RouterHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ManifestPath string `json:"manifest_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ManifestPath == "" {
			req.ManifestPath = "configs/router.example.yaml"
		}
		if _, err := RunManifestReport(req.ManifestPath); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	mux.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ManifestPath string `json:"manifest_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ManifestPath == "" {
			req.ManifestPath = "configs/router.example.yaml"
		}
		if err := ValidateManifest(req.ManifestPath); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/replay", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TracePath string `json:"trace_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.TracePath == "" {
			http.Error(w, "trace_path is required", http.StatusBadRequest)
			return
		}
		if err := ReplayTrace(req.TracePath, w); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	})
	return mux
}

func StartRouterServer(ctx context.Context, addr string) error {
	if addr == "" {
		addr = ":8080"
	}
	s := &http.Server{Addr: addr, Handler: RouterHandler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()
	return s.ListenAndServe()
}

func StartRouterServerTLS(ctx context.Context, addr string, certFile string, keyFile string, caFile string, requireClientCert bool) error {
	if addr == "" {
		addr = ":8080"
	}
	cfg, err := security.BuildServerTLSConfig(certFile, keyFile, caFile, requireClientCert)
	if err != nil {
		return err
	}
	s := &http.Server{Addr: addr, Handler: RouterHandler(), ReadHeaderTimeout: 5 * time.Second, TLSConfig: cfg}
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()
	ln, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("router tls listen: %w", err)
	}
	return s.Serve(ln)
}

func StartRouterServerFromEnv(ctx context.Context) error {
	addr := os.Getenv("ROUTER_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if envBool("ROUTER_TLS_ENABLED") {
		return StartRouterServerTLS(
			ctx,
			addr,
			os.Getenv("ROUTER_TLS_CERT_FILE"),
			os.Getenv("ROUTER_TLS_KEY_FILE"),
			os.Getenv("ROUTER_TLS_CA_FILE"),
			envBool("ROUTER_TLS_REQUIRE_CLIENT_CERT"),
		)
	}
	return StartRouterServer(ctx, addr)
}

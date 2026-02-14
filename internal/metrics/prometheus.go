package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/your-org/agent-router/internal/security"
)

// PrometheusRecorder reports runtime metrics using Prometheus primitives.
type PrometheusRecorder struct {
	invocations *prometheus.CounterVec
	durations   *prometheus.HistogramVec
	retries     *prometheus.CounterVec
	circuitOpen *prometheus.CounterVec
}

func NewPrometheusRecorder(registry *prometheus.Registry) (*PrometheusRecorder, error) {
	if registry == nil {
		return nil, fmt.Errorf("prometheus registry is nil")
	}

	r := &PrometheusRecorder{
		invocations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "agent_router_invocations_total",
			Help: "Total number of agent invocations by status",
		}, []string{"agent_id", "status"}),
		durations: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "agent_router_invocation_duration_seconds",
			Help:    "Agent invocation latency in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"agent_id"}),
		retries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "agent_router_retry_attempts_total",
			Help: "Total retry attempts by agent",
		}, []string{"agent_id"}),
		circuitOpen: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "agent_router_circuit_breaks_total",
			Help: "Total circuit breaker open events by agent",
		}, []string{"agent_id"}),
	}

	for _, collector := range []prometheus.Collector{r.invocations, r.durations, r.retries, r.circuitOpen} {
		if err := registry.Register(collector); err != nil {
			return nil, fmt.Errorf("register collector: %w", err)
		}
	}
	return r, nil
}

func (r *PrometheusRecorder) ObserveInvocation(agentID string, status string, duration time.Duration) {
	r.invocations.WithLabelValues(agentID, status).Inc()
	r.durations.WithLabelValues(agentID).Observe(duration.Seconds())
}

func (r *PrometheusRecorder) ObserveRetry(agentID string) {
	r.retries.WithLabelValues(agentID).Inc()
}

func (r *PrometheusRecorder) ObserveCircuitOpen(agentID string) {
	r.circuitOpen.WithLabelValues(agentID).Inc()
}

func StartPrometheusServer(addr string, registry *prometheus.Registry) (*http.Server, error) {
	if addr == "" {
		addr = ":2112"
	}
	if registry == nil {
		return nil, fmt.Errorf("prometheus registry is nil")
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen metrics endpoint %q: %w", addr, err)
	}

	srv := &http.Server{
		Addr:    ln.Addr().String(),
		Handler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	return srv, nil
}

// StartPrometheusServerTLS starts metrics endpoint with optional client-cert auth (mTLS).
func StartPrometheusServerTLS(addr string, registry *prometheus.Registry, certFile string, keyFile string, caFile string, requireClientCert bool) (*http.Server, error) {
	if addr == "" {
		addr = ":2112"
	}
	if registry == nil {
		return nil, fmt.Errorf("prometheus registry is nil")
	}

	tlsCfg, err := security.BuildServerTLSConfig(certFile, keyFile, caFile, requireClientCert)
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen metrics endpoint %q: %w", addr, err)
	}
	tlsListener := tls.NewListener(ln, tlsCfg)

	srv := &http.Server{
		Addr:    ln.Addr().String(),
		Handler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}
	go func() {
		_ = srv.Serve(tlsListener)
	}()
	return srv, nil
}

func StopServer(ctx context.Context, srv *http.Server) error {
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

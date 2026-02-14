package unit

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/your-org/fluxroute/internal/metrics"
)

func TestPrometheusRecorderAndEndpoint(t *testing.T) {
	reg := prometheus.NewRegistry()
	rec, err := metrics.NewPrometheusRecorder(reg)
	if err != nil {
		t.Fatalf("new prometheus recorder: %v", err)
	}

	rec.ObserveInvocation("agent_a", "success", 10*time.Millisecond)
	rec.ObserveRetry("agent_a")
	rec.ObserveCircuitOpen("agent_a")

	srv, err := metrics.StartPrometheusServer("127.0.0.1:0", reg)
	if err != nil {
		t.Fatalf("start metrics server: %v", err)
	}
	defer func() { _ = metrics.StopServer(context.Background(), srv) }()

	resp, err := http.Get("http://" + srv.Addr)
	if err != nil {
		t.Fatalf("GET metrics endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "fluxroute_invocations_total") {
		t.Fatalf("missing invocations metric: %s", text)
	}
	if !strings.Contains(text, "fluxroute_circuit_breaks_total") {
		t.Fatalf("missing circuit metric: %s", text)
	}
}

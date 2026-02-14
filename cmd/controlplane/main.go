package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/your-org/fluxroute/internal/controlplane"
	"github.com/your-org/fluxroute/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(version.String())
		return
	}

	addr := os.Getenv("CONTROLPLANE_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	svc := controlplane.NewService()
	tlsEnabled := os.Getenv("CONTROLPLANE_TLS_ENABLED") == "true"
	var err error
	if tlsEnabled {
		err = controlplane.StartServerTLS(
			ctx,
			addr,
			svc,
			os.Getenv("CONTROLPLANE_TLS_CERT_FILE"),
			os.Getenv("CONTROLPLANE_TLS_KEY_FILE"),
			os.Getenv("CONTROLPLANE_TLS_CA_FILE"),
			os.Getenv("CONTROLPLANE_TLS_REQUIRE_CLIENT_CERT") == "true",
		)
	} else {
		err = controlplane.StartServer(ctx, addr, svc)
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "controlplane failed: %v\n", err)
		os.Exit(1)
	}
}

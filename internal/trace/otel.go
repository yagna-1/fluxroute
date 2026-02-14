package trace

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OTelRuntime stores initialized tracer and shutdown hook.
type OTelRuntime struct {
	Tracer   oteltrace.Tracer
	Shutdown func(context.Context) error
}

// SetupOTelFromEnv initializes OpenTelemetry when TRACE_ENABLED=true.
func SetupOTelFromEnv(serviceName string) (OTelRuntime, error) {
	noop := OTelRuntime{
		Tracer:   otel.Tracer(serviceName),
		Shutdown: func(context.Context) error { return nil },
	}

	if !envBool("TRACE_ENABLED") {
		return noop, nil
	}

	ctx := context.Background()
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return OTelRuntime{}, fmt.Errorf("otel resource: %w", err)
	}

	var exp sdktrace.SpanExporter
	endpoint := strings.TrimSpace(os.Getenv("TRACE_ENDPOINT"))
	if endpoint != "" {
		exp, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return OTelRuntime{}, fmt.Errorf("otel otlp exporter: %w", err)
		}
	} else {
		exp, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return OTelRuntime{}, fmt.Errorf("otel stdout exporter: %w", err)
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return OTelRuntime{
		Tracer:   tp.Tracer(serviceName),
		Shutdown: tp.Shutdown,
	}, nil
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

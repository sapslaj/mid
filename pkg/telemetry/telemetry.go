package telemetry

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/sapslaj/mid/pkg/env"
)

func StartTelemetry() func() {
	// NOTE: Telemetry is _ONLY_ set up if `PULUMI_MID_OTLP_ENDPOINT` is set.
	// There is _NO_ default value for this. This means that this telemetry is
	// *OPT-IN* and only if you have an OTLP endpoint to send things to.
	// I (@sapslaj) vow to never do opt-out telemetry of my own will.

	endpoint, err := env.GetDefault("PULUMI_MID_OTLP_ENDPOINT", "")
	if err != nil {
		slog.Error("error getting otel OTLP endpoint", slog.Any("error", err))
		return func() {}
	}
	if endpoint == "" {
		slog.Info("telemetry disabled")
		return func() {}
	}

	ctx := context.Background()
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(endpoint),
		),
	)
	if err != nil {
		slog.Error("error setting up otlptrace", slog.Any("error", err))
		return func() {}
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", "pulumi-resource-mid"),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		slog.Error("error setting up otel resource", slog.Any("error", err))
		return func() {}
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		),
	)

	slog.Info("started telemetry", slog.String("otel_endpoint", endpoint))

	return func() {
		slog.Info("stopping telemetry", slog.String("otel_endpoint", endpoint))
		time.Sleep(10 * time.Second)
		exporter.Shutdown(ctx)
		slog.Info("telemetry stopped", slog.String("otel_endpoint", endpoint))
	}
}

func SlogJSON(key string, value any) slog.Attr {
	data, err := json.Marshal(value)
	if err != nil {
		return slog.String(key, "err!"+err.Error())
	}
	return slog.String(key, string(data))
}

func OtelJSON(key string, value any) attribute.KeyValue {
	data, err := json.Marshal(value)
	if err != nil {
		return attribute.String(key, "err!"+err.Error())
	}
	return attribute.String(key, string(data))
}

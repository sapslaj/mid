// Copyright 2016-2023, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log/slog"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/sapslaj/mid/pkg/env"
	mid "github.com/sapslaj/mid/provider"
	"github.com/sapslaj/mid/version"
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

	return func() {
		time.Sleep(time.Second)
		exporter.Shutdown(ctx)
	}
}

// Serve the provider against Pulumi's Provider protocol.
func main() {
	shutdownTelemetry := StartTelemetry()
	defer shutdownTelemetry()
	p.RunProvider(mid.Name, version.Version, mid.Provider())
}

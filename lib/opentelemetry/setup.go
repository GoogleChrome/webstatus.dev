// Copyright 2024 Google LLC
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

package opentelemetry

import (
	"context"
	"errors"
	"log"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Instruments the application with OpenTelemetry and exports OTLP for
// metrics and traces. It uses log/slog for logging, and writes logs to stdout.
// The application does not include any GCP dependencies, and instead only uses
// open standards for instrumentation. The OpenTelemetry collector is used to
// route telemetry to GCP.
// Afterwards, metrics, tracing, logging, and context propagation will be setup.
func SetupOpenTelemetry(ctx context.Context, projectID string) (shutdown func(context.Context) error, err error) {
	setupLogging(projectID)

	var shutdownFuncs []func(context.Context) error

	// Set up the Cloud Trace exporter.
	exporter, err := cloudtrace.New()
	if err != nil {
		return nil, err
	}

	// Identify your application using resource detection.
	res, err := resource.New(ctx,
		// Use the GCP resource detector to detect information about the GKE Cluster.
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		log.Fatalf("resource.New: %v", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	// shutdown combines shutdown functions from multiple OpenTelemetry
	// components into a single function.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil

		return err
	}

	// Configure Context Propagation to use the default W3C traceparent format
	// Set the global Propagators which is used by otelhttp to propagate
	// context using the w3c traceparent and baggage formats.
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	// Configure Trace Export to send spans as OTLP
	// texporter, err := autoexport.NewSpanExporter(ctx)
	// if err != nil {
	// 	err = errors.Join(err, shutdown(ctx))

	// 	return nil, err
	// }
	// tp := trace.NewTracerProvider(trace.WithBatcher(texporter))
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	otel.SetTracerProvider(tp)

	// Configure Metric Export to send metrics as OTLP
	mreader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		err = errors.Join(err, shutdown(ctx))

		return nil, err
	}
	mp := metric.NewMeterProvider(
		metric.WithReader(mreader),
	)
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
	otel.SetMeterProvider(mp)

	return shutdown, nil
}

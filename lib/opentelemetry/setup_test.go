// Copyright 2026 Google LLC
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
	"testing"
)

func TestMaybeSetup_Disabled(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")

	shutdown, err := MaybeSetup(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("expected nil error from shutdown, got: %v", err)
	}
}

func TestMaybeSetup_MissingProjectID(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "webstatus-test")
	t.Setenv("OTEL_GCP_PROJECT_ID", "")

	shutdown, err := MaybeSetup(context.Background())
	if err == nil {
		t.Fatal("expected error for missing OTEL_GCP_PROJECT_ID, got nil")
	}
	if shutdown != nil {
		t.Fatal("expected nil shutdown function on error")
	}
}

func TestSetupOpenTelemetry(t *testing.T) {
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	t.Setenv("OTEL_METRICS_EXPORTER", "none")

	ctx := context.Background()
	shutdown, err := SetupOpenTelemetry(ctx, "test-project")
	if err != nil {
		t.Fatalf("SetupOpenTelemetry failed: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

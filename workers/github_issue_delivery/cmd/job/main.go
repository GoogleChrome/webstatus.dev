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

package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/GoogleChrome/webstatus.dev/lib/gcppubsub"
	"github.com/GoogleChrome/webstatus.dev/lib/gcppubsub/gcppubsubadapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/GoogleChrome/webstatus.dev/lib/opentelemetry"
	"github.com/GoogleChrome/webstatus.dev/workers/github_issue_delivery/pkg/delivery"
	"github.com/google/uuid"
)

func main() {
	ctx := context.Background()

	shutdown, err := opentelemetry.MaybeSetup(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to setup opentelemetry", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "unable to shutdown opentelemetry", "error", err)
		}
	}()

	slog.InfoContext(ctx, "starting GitHub issue delivery worker")

	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		slog.ErrorContext(ctx, "PROJECT_ID is not set. exiting...")
		os.Exit(1)
	}

	subID := os.Getenv("GITHUB_ISSUE_DELIVERY_SUBSCRIPTION_ID")
	if subID == "" {
		slog.ErrorContext(ctx, "GITHUB_ISSUE_DELIVERY_SUBSCRIPTION_ID is not set. exiting...")
		os.Exit(1)
	}

	spannerDB := os.Getenv("SPANNER_DATABASE")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create spanner client", "error", err.Error())
		os.Exit(1)
	}

	if _, found := os.LookupEnv("SPANNER_EMULATOR_HOST"); found {
		slog.InfoContext(ctx, "setting spanner to local mode")
		spannerClient.SetFeatureSearchBaseQuery(gcpspanner.LocalFeatureBaseQuery{})
		spannerClient.SetMisingOneImplementationQuery(gcpspanner.LocalMissingOneImplementationQuery{})
	}

	appID := os.Getenv("GITHUB_APP_ID")
	pkPath := os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")

	var privateKeyPEM []byte
	if pkPath != "" {
		pkData, err := os.ReadFile(filepath.Clean(pkPath)) // #nosec G304 G703 -- Admin configured private key path
		if err != nil {
			slog.WarnContext(ctx, "unable to read private key file", "path", pkPath, "error", err)
		} else {
			privateKeyPEM = pkData
		}
	}

	tokenProvider, err := gh.NewTokenProvider(appID, privateKeyPEM, nil)
	if err != nil {
		slog.WarnContext(ctx, "unable to create token provider", "error", err)
	}
	_ = tokenProvider

	queueClient, err := gcppubsub.NewClient(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create pub sub client", "error", err)
		os.Exit(1)
	}

	workerID := uuid.NewString()
	ghClient := gh.NewClient("")
	deliverer := delivery.NewDeliverer(ghClient, spannerClient, workerID)

	listener := gcppubsubadapters.NewGitHubIssueDeliverySubscriberAdapter(deliverer, queueClient, subID)
	if err := listener.Subscribe(ctx); err != nil {
		slog.ErrorContext(ctx, "github issue delivery worker subscriber failed", "error", err)
		os.Exit(1)
	}
}

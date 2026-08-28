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
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/vcs_sync/pkg/workflow"
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

	projectID := os.Getenv("PROJECT_ID")
	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	spannerDB := os.Getenv("SPANNER_DATABASE")
	scanTasksTopic := os.Getenv("VCS_SCAN_TASKS_TOPIC")
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

	spannerClient, err := gcpspanner.NewSpannerClient(projectID, spannerInstance, spannerDB)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create spanner client", "error", err)
		os.Exit(1)
	}

	pubsubClient, err := gcppubsub.NewClient(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create pubsub client", "error", err)
		os.Exit(1)
	}

	publisher := gcppubsubadapters.NewVCSSyncPublisherAdapter(pubsubClient, scanTasksTopic)
	cacher := gh.NewLocalTokenCacher()
	tokenProvider, err := gh.NewTokenProvider(appID, privateKeyPEM, cacher)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create token provider", "error", err)
		os.Exit(1)
	}
	repoLister := workflow.DefaultGitHubRepoLister{}

	processor := workflow.NewVCSSyncProcessor(spannerClient, tokenProvider, repoLister, publisher)
	if err := processor.Process(ctx, workflow.NewJobArguments()); err != nil {
		slog.ErrorContext(ctx, "scheduled VCS sync failed", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "scheduled VCS sync job finished successfully")
}

// Copyright 2025 Google LLC
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
	"net/url"
	"os"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/email/chime"
	"github.com/GoogleChrome/webstatus.dev/lib/email/chime/chimeadapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gcppubsub"
	"github.com/GoogleChrome/webstatus.dev/lib/gcppubsub/gcppubsubadapters"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters"
	"github.com/GoogleChrome/webstatus.dev/workers/email/pkg/digest"
	"github.com/GoogleChrome/webstatus.dev/workers/email/pkg/sender"
)

func main() {
	ctx := context.Background()

	slog.InfoContext(ctx, "starting email worker")

	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		slog.ErrorContext(ctx, "PROJECT_ID is not set. exiting...")
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

	baseURL := os.Getenv("FRONTEND_BASE_URL")
	if baseURL == "" {
		slog.ErrorContext(ctx, "FRONTEND_BASE_URL is not set. exiting...")
		os.Exit(1)
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse FRONTEND_BASE_URL", "error", err.Error())
		os.Exit(1)
	}

	// For subscribing to email events
	emailSubID := os.Getenv("EMAIL_SUBSCRIPTION_ID")
	if emailSubID == "" {
		slog.ErrorContext(ctx, "EMAIL_SUBSCRIPTION_ID is not set. exiting...")
		os.Exit(1)
	}

	queueClient, err := gcppubsub.NewClient(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create pub sub client", "error", err)
		os.Exit(1)
	}

	renderer, err := digest.NewHTMLRenderer(parsedBaseURL.String())
	if err != nil {
		// If the template is not valid, the renderer will fail.
		slog.ErrorContext(ctx, "unable to create renderer", "error", err)
		os.Exit(1)
	}

	var emailSender sender.EmailSender

	slog.InfoContext(ctx, "using chime email sender")
	chimeEnvStr := os.Getenv("CHIME_ENV")
	chimeEnv := chime.EnvProd
	if chimeEnvStr == "autopush" {
		chimeEnv = chime.EnvAutopush
	}
	chimeBCC := os.Getenv("CHIME_BCC")
	bccList := []string{}
	if chimeBCC != "" {
		bccList = strings.Split(chimeBCC, ",")
	}
	fromAddress := os.Getenv("FROM_ADDRESS")
	chimeSender, err := chime.NewChimeSender(ctx, chimeEnv, bccList, fromAddress, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create chime sender", "error", err)
		os.Exit(1)
	}
	emailSender = chimeadapters.NewEmailWorkerChimeAdapter(chimeSender)

	listener := gcppubsubadapters.NewEmailWorkerSubscriberAdapter(sender.NewSender(
		emailSender,
		spanneradapters.NewEmailWorkerChannelStateManager(spannerClient),
		renderer,
	), queueClient, emailSubID)
	if err := listener.Subscribe(ctx); err != nil {
		slog.ErrorContext(ctx, "worker subscriber failed", "error", err)
		os.Exit(1)
	}
}

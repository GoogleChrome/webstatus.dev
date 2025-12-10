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
	"os"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpgcs"
	"github.com/GoogleChrome/webstatus.dev/lib/gcppubsub"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

func main() {
	ctx := context.Background()

	slog.InfoContext(ctx, "starting event producer worker")

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

	// For subscribing to ingestion events
	ingestionSubID := os.Getenv("INGESTION_SUBSCRIPTION_ID")
	if ingestionSubID == "" {
		slog.ErrorContext(ctx, "INGESTION_SUBSCRIPTION_ID is not set. exiting...")
		os.Exit(1)
	}

	// For publishing to notification events
	notificationTopicID := os.Getenv("NOTIFICATION_TOPIC_ID")
	if notificationTopicID == "" {
		slog.ErrorContext(ctx, "NOTIFICATION_TOPIC_ID is not set. exiting...")
		os.Exit(1)
	}

	stateBlobBucket := os.Getenv("STATE_BLOB_BUCKET")
	if stateBlobBucket == "" {
		slog.ErrorContext(ctx, "STATE_BLOB_BUCKET is not set. exiting...")
		os.Exit(1)
	}

	queueClient, err := gcppubsub.NewClient(ctx, projectID)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create pub sub client", "error", err)
		os.Exit(1)
	}

	_, err = gcpgcs.NewClient(ctx, stateBlobBucket)
	if err != nil {
		slog.ErrorContext(ctx, "unable to create gcs client", "error", err)
		os.Exit(1)
	}

	// Nil handler for now. Will fix later
	err = queueClient.Subscribe(ctx, ingestionSubID, nil)
	if err != nil {
		slog.ErrorContext(ctx, "unable to connect to subscription", "error", err)
		os.Exit(1)
	}
}

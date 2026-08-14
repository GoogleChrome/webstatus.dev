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

package spanneradapters

import (
	"context"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

// VCSScannerSpannerClient defines the database interface required by the VCS scanner adapter.
type VCSScannerSpannerClient interface {
	SynchronizeRepositoryCodeSubscriptions(
		ctx context.Context,
		vcsProvider, vcsInstallationID, vcsRepositoryID string,
		subscriptions []gcpspanner.CodeSubscription,
		now time.Time,
	) error
	InsertCodeSubscriptionScanLog(ctx context.Context, scanLog gcpspanner.CodeSubscriptionScanLog) error
}

// VCSScannerAdapter adapts domain scan results to Spanner mutations.
type VCSScannerAdapter struct {
	client VCSScannerSpannerClient
}

// NewVCSScannerAdapter creates a new VCSScannerAdapter.
func NewVCSScannerAdapter(client VCSScannerSpannerClient) *VCSScannerAdapter {
	return &VCSScannerAdapter{client: client}
}

// SynchronizeScanResult adapts and persists a domain ScanResult into Spanner.
func (a *VCSScannerAdapter) SynchronizeScanResult(
	ctx context.Context,
	vcsProvider, vcsInstallationID, vcsRepositoryID string,
	result *codescan.ScanResult,
	now time.Time,
) error {
	spannerSubs := make([]gcpspanner.CodeSubscription, 0, len(result.Subscriptions))
	for _, sub := range result.Subscriptions {
		occurrences := make([]gcpspanner.SubscriptionOccurrence, 0, len(sub.Occurrences))
		for _, occ := range sub.Occurrences {
			occurrences = append(occurrences, gcpspanner.SubscriptionOccurrence{
				FilePath:       occ.FilePath,
				LineNumber:     occ.LineNumber,
				CommentSnippet: occ.CommentSnippet,
			})
		}

		spannerSubs = append(spannerSubs, gcpspanner.CodeSubscription{
			ID:                 sub.ID,
			VCSProvider:        gcpspanner.VCSProvider(sub.VCSProvider),
			VCSInstallationID:  sub.VCSInstallationID,
			VCSRepositoryID:    sub.VCSRepositoryID,
			RepositoryFullName: sub.RepositoryFullName,
			TargetQuery:        sub.TargetQuery,
			Triggers:           sub.Triggers,
			Status:             gcpspanner.SubscriptionActive,
			Occurrences:        occurrences,
			CreatedAt:          sub.CreatedAt,
			UpdatedAt:          sub.UpdatedAt,
		})
	}

	return a.client.SynchronizeRepositoryCodeSubscriptions(
		ctx,
		vcsProvider,
		vcsInstallationID,
		vcsRepositoryID,
		spannerSubs,
		now,
	)
}

// RecordScanLog adapts and persists a commit scan log into Spanner.
func (a *VCSScannerAdapter) RecordScanLog(
	ctx context.Context,
	log gcpspanner.CodeSubscriptionScanLog,
) error {
	return a.client.InsertCodeSubscriptionScanLog(ctx, log)
}

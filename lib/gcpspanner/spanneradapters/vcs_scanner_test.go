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
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/codescan"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

type mockVCSScannerSpannerClient struct {
	syncCalled bool
	logCalled  bool
	passedSubs []gcpspanner.CodeSubscriptionInput
	passedLog  gcpspanner.CodeSubscriptionScanLog
}

func (m *mockVCSScannerSpannerClient) SynchronizeRepositoryCodeSubscriptions(
	_ context.Context,
	_ gcpspanner.VCSProvider,
	_ string,
	subscriptions []gcpspanner.CodeSubscriptionInput,
) error {
	m.syncCalled = true
	m.passedSubs = subscriptions

	return nil
}

func (m *mockVCSScannerSpannerClient) InsertCodeSubscriptionScanLog(
	_ context.Context,
	log gcpspanner.CodeSubscriptionScanLog,
) error {
	m.logCalled = true
	m.passedLog = log

	return nil
}

func TestVCSScannerAdapter_SynchronizeScanResult(t *testing.T) {
	t.Parallel()

	mockClient := &mockVCSScannerSpannerClient{
		syncCalled: false,
		logCalled:  false,
		passedSubs: nil,
		passedLog: gcpspanner.CodeSubscriptionScanLog{
			ID:              "",
			VCSProvider:     "",
			VCSRepositoryID: "",
			CommitSHA:       "",
			Branch:          "",
			ScanStatus:      "",
			FilesScanned:    0,
			DirectivesFound: 0,
			ErrorMessage:    nil,
			ScannedAt:       time.Time{},
		},
	}
	adapter := NewVCSScannerAdapter(mockClient)

	scanResult := &codescan.ScanResult{
		Warnings: nil,
		Subscriptions: []codescan.ScannedSubscription{
			{
				VCSProvider:        "github",
				VCSInstallationID:  "inst-1",
				VCSRepositoryID:    "repo-1",
				RepositoryFullName: "owner/repo",
				TargetQuery:        "id:view-transitions",
				Triggers: []codescan.SubscriptionTrigger{
					codescan.SubscriptionTriggerFeatureBaselinePromoteToWidely,
				},
				Occurrences: []codescan.SubscriptionOccurrence{
					{
						FilePath:       "src/app.ts",
						LineNumber:     10,
						CommentSnippet: "// TODO(baseline/view-transitions)",
					},
				},
			},
		},
		FilesScanned:    1,
		BytesScanned:    100,
		DirectivesFound: 1,
		IsTruncated:     false,
		ScanStatus:      codescan.ScanStatusSuccess,
	}

	err := adapter.SynchronizeScanResult(
		context.Background(),
		"github",
		"inst-1",
		"repo-1",
		scanResult,
	)
	if err != nil {
		t.Fatalf("SynchronizeScanResult returned error: %v", err)
	}

	if !mockClient.syncCalled {
		t.Errorf("expected SynchronizeRepositoryCodeSubscriptions to be called")
	}
	if len(mockClient.passedSubs) != 1 {
		t.Fatalf("expected 1 subscription passed, got %d", len(mockClient.passedSubs))
	}

	sub := mockClient.passedSubs[0]
	if sub.TargetQuery != "id:view-transitions" {
		t.Errorf("unexpected target query: %s", sub.TargetQuery)
	}
	if len(sub.Occurrences) != 1 || sub.Occurrences[0].LineNumber != 10 {
		t.Errorf("unexpected occurrences: %+v", sub.Occurrences)
	}
}

func TestVCSScannerAdapter_RecordScanLog(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	mockClient := &mockVCSScannerSpannerClient{
		syncCalled: false,
		logCalled:  false,
		passedSubs: nil,
		passedLog: gcpspanner.CodeSubscriptionScanLog{
			ID:              "",
			VCSProvider:     "",
			VCSRepositoryID: "",
			CommitSHA:       "",
			Branch:          "",
			ScanStatus:      "",
			FilesScanned:    0,
			DirectivesFound: 0,
			ErrorMessage:    nil,
			ScannedAt:       time.Time{},
		},
	}
	adapter := NewVCSScannerAdapter(mockClient)

	log := gcpspanner.CodeSubscriptionScanLog{
		ID:              "log-1",
		VCSProvider:     "github",
		VCSRepositoryID: "repo-1",
		CommitSHA:       "sha-123",
		Branch:          "main",
		ScanStatus:      gcpspanner.ScanStatusSuccess,
		FilesScanned:    5,
		DirectivesFound: 2,
		ErrorMessage:    nil,
		ScannedAt:       now,
	}

	err := adapter.RecordScanLog(context.Background(), log)
	if err != nil {
		t.Fatalf("RecordScanLog returned error: %v", err)
	}

	if !mockClient.logCalled {
		t.Errorf("expected InsertCodeSubscriptionScanLog to be called")
	}
	if mockClient.passedLog.ID != "log-1" {
		t.Errorf("unexpected log passed: %+v", mockClient.passedLog)
	}
}

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

package gcppubsubadapters

import (
	"context"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	githubissuedeliveryv1 "github.com/GoogleChrome/webstatus.dev/lib/event/githubissuedelivery/v1"
)

type mockGitHubIssueDeliverer struct {
	lastJob githubissuedeliveryv1.GitHubIssueDeliveryEvent
	called  bool
}

func (m *mockGitHubIssueDeliverer) ProcessJob(
	_ context.Context,
	job githubissuedeliveryv1.GitHubIssueDeliveryEvent,
) error {
	m.called = true
	m.lastJob = job

	return nil
}

func TestGitHubIssueDeliverySubscriberAdapter_Subscribe(t *testing.T) {
	t.Parallel()

	deliverer := &mockGitHubIssueDeliverer{
		lastJob: githubissuedeliveryv1.GitHubIssueDeliveryEvent{
			DeliveryID:         "",
			SubscriptionID:     "",
			VCSProvider:        "",
			VCSInstallationID:  "",
			VCSRepositoryID:    "",
			RepositoryOwner:    "",
			RepositoryName:     "",
			RepositoryFullName: "",
			FeatureID:          "",
			FeatureName:        "",
			Trigger:            "",
			CommitSHA:          "",
			Occurrences:        nil,
			WebStatusURL:       "",
		},
		called: false,
	}
	subscriber := &mockEventSubscriber{
		subscribedSubID: "",
		handler:         nil,
	}

	adapter := NewGitHubIssueDeliverySubscriberAdapter(deliverer, subscriber, "test-github-delivery-sub")

	ctx := context.Background()
	if err := adapter.Subscribe(ctx); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if subscriber.subscribedSubID != "test-github-delivery-sub" {
		t.Errorf("expected subscription ID 'test-github-delivery-sub', got %s", subscriber.subscribedSubID)
	}

	data, err := event.New(githubissuedeliveryv1.GitHubIssueDeliveryEvent{
		DeliveryID:         "del-123",
		SubscriptionID:     "sub-456",
		VCSProvider:        "github",
		VCSInstallationID:  "inst-789",
		VCSRepositoryID:    "repo-999",
		RepositoryOwner:    "GoogleChrome",
		RepositoryName:     "webstatus.dev",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		FeatureID:          "grid",
		FeatureName:        "CSS Grid",
		Trigger:            "feature.baseline.promote_to_newly",
		CommitSHA:          "abcdef123456",
		Occurrences: []githubissuedeliveryv1.IssueOccurrence{
			{
				FilePath:       "styles/main.css",
				LineNumber:     42,
				CommentSnippet: "// TODO: Grid",
			},
		},
		WebStatusURL: "https://webstatus.dev/features/grid",
	})
	if err != nil {
		t.Fatalf("event.New failed: %v", err)
	}

	if err := subscriber.handler(ctx, "msg-delivery-1", data); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if !deliverer.called {
		t.Errorf("expected deliverer to be called")
	}
	if deliverer.lastJob.DeliveryID != "del-123" {
		t.Errorf("expected DeliveryID 'del-123', got %s", deliverer.lastJob.DeliveryID)
	}
}

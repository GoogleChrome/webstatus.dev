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
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
)

type mockVCSScannerTaskProcessor struct {
	lastTask codescantaskv1.CodeScanTaskEvent
	called   bool
}

func (m *mockVCSScannerTaskProcessor) ProcessTask(_ context.Context, task codescantaskv1.CodeScanTaskEvent) error {
	m.called = true
	m.lastTask = task

	return nil
}

type mockEventSubscriber struct {
	subscribedSubID string
	handler         func(ctx context.Context, msgID string, data []byte) error
}

func (m *mockEventSubscriber) Subscribe(_ context.Context, subscriptionID string,
	handler func(ctx context.Context, msgID string, data []byte) error) error {
	m.subscribedSubID = subscriptionID
	m.handler = handler

	return nil
}

func TestVCSScannerSubscriberAdapter_Subscribe(t *testing.T) {
	t.Parallel()

	processor := &mockVCSScannerTaskProcessor{
		lastTask: codescantaskv1.CodeScanTaskEvent{
			VCSProvider:        "",
			VCSInstallationID:  "",
			VCSRepositoryID:    "",
			RepositoryFullName: "",
			CommitSHA:          "",
			Branch:             "",
			IsDefaultBranch:    false,
			ModifiedFiles:      nil,
		},
		called: false,
	}
	subscriber := &mockEventSubscriber{
		subscribedSubID: "",
		handler:         nil,
	}

	adapter := NewVCSScannerSubscriberAdapter(processor, subscriber, "test-vcs-scan-sub")

	ctx := context.Background()
	if err := adapter.Subscribe(ctx); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if subscriber.subscribedSubID != "test-vcs-scan-sub" {
		t.Errorf("expected subscription ID 'test-vcs-scan-sub', got %s", subscriber.subscribedSubID)
	}

	data, err := event.New(codescantaskv1.CodeScanTaskEvent{
		VCSProvider:        "github",
		VCSInstallationID:  "inst-1",
		VCSRepositoryID:    "repo-1",
		RepositoryFullName: "owner/repo",
		CommitSHA:          "sha123",
		Branch:             "main",
		IsDefaultBranch:    true,
		ModifiedFiles:      nil,
	})
	if err != nil {
		t.Fatalf("event.New failed: %v", err)
	}

	if err := subscriber.handler(ctx, "msg-123", data); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if !processor.called {
		t.Errorf("expected processor to be called")
	}
	if processor.lastTask.RepositoryFullName != "owner/repo" {
		t.Errorf("expected repository full name 'owner/repo', got %s", processor.lastTask.RepositoryFullName)
	}
}

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
	"encoding/json"
	"errors"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/google/go-cmp/cmp"
)

func TestVCSSyncPublisherAdapter_PublishCodeScanTask(t *testing.T) {
	t.Parallel()

	task := codescantaskv1.CodeScanTaskEvent{
		VCSProvider:        "github",
		VCSInstallationID:  "inst-123",
		VCSRepositoryID:    "repo-456",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		CommitSHA:          "abcdef123456",
		Branch:             "main",
		IsDefaultBranch:    true,
		ModifiedFiles:      []string{"src/app.ts"},
	}

	expectedEnvelope, err := event.New(task)
	if err != nil {
		t.Fatalf("failed to create expected envelope: %v", err)
	}

	tests := []struct {
		name       string
		publishErr error
		wantErr    bool
	}{
		{
			name:       "success",
			publishErr: nil,
			wantErr:    false,
		},
		{
			name:       "publisher error",
			publishErr: errors.New("pubsub connection failure"),
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			publisher := new(mockPublisher)
			publisher.err = tc.publishErr
			adapter := NewVCSSyncPublisherAdapter(publisher, "vcs-scan-topic")

			err := adapter.PublishCodeScanTask(context.Background(), task)
			if (err != nil) != tc.wantErr {
				t.Errorf("PublishCodeScanTask() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if publisher.publishedTopic != "vcs-scan-topic" {
				t.Errorf("Topic mismatch: got %s, want vcs-scan-topic", publisher.publishedTopic)
			}

			var actual any
			if err := json.Unmarshal(publisher.publishedData, &actual); err != nil {
				t.Fatalf("failed to unmarshal published data: %v", err)
			}

			var expected any
			if err := json.Unmarshal(expectedEnvelope, &expected); err != nil {
				t.Fatalf("failed to unmarshal expected data: %v", err)
			}

			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("Payload mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

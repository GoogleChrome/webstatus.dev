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

package v1_test

import (
	"encoding/json"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	githubissuedeliveryv1 "github.com/GoogleChrome/webstatus.dev/lib/event/githubissuedelivery/v1"
)

func TestGitHubIssueDeliveryEvent_EventMetadataAndSerialization(t *testing.T) {
	t.Parallel()

	evt := githubissuedeliveryv1.GitHubIssueDeliveryEvent{
		DeliveryID:         "del-123",
		SubscriptionID:     "sub-456",
		VCSProvider:        "github",
		VCSInstallationID:  "inst-789",
		VCSRepositoryID:    "repo-101",
		RepositoryOwner:    "GoogleChrome",
		RepositoryName:     "webstatus.dev",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		FeatureID:          "subgrid",
		FeatureName:        "CSS Subgrid",
		Trigger:            "feature_baseline_to_widely",
		CommitSHA:          "abcdef",
		Occurrences:        nil,
		WebStatusURL:       "https://webstatus.dev/features/subgrid",
	}

	if evt.Kind() != "GitHubIssueDeliveryEvent" {
		t.Errorf("Kind() = %s, want GitHubIssueDeliveryEvent", evt.Kind())
	}
	if evt.APIVersion() != "v1" {
		t.Errorf("APIVersion() = %s, want v1", evt.APIVersion())
	}

	envelopeBytes, err := event.New(evt)
	if err != nil {
		t.Fatalf("event.New failed: %v", err)
	}

	var env struct {
		Kind       string          `json:"kind"`
		APIVersion string          `json:"apiVersion"`
		Data       json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(envelopeBytes, &env); err != nil {
		t.Fatalf("json.Unmarshal envelope failed: %v", err)
	}

	if env.Kind != "GitHubIssueDeliveryEvent" || env.APIVersion != "v1" {
		t.Errorf("unexpected envelope headers: %+v", env)
	}

	var parsed githubissuedeliveryv1.GitHubIssueDeliveryEvent
	if err := json.Unmarshal(env.Data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal inner data failed: %v", err)
	}
	if parsed.DeliveryID != "del-123" {
		t.Errorf("parsed.DeliveryID = %s, want del-123", parsed.DeliveryID)
	}
}

func TestDeriveDeliveryID_DeterminismAndUniqueness(t *testing.T) {
	t.Parallel()

	subID1 := "sub-123"
	subID2 := "sub-456"
	trigger1 := "feature_baseline_to_newly"
	trigger2 := "feature_baseline_to_widely"

	id1 := githubissuedeliveryv1.DeriveDeliveryID(subID1, trigger1)
	id2 := githubissuedeliveryv1.DeriveDeliveryID(subID1, trigger1)
	if id1 != id2 {
		t.Errorf("expected deterministic IDs for identical inputs, got %s vs %s", id1, id2)
	}

	idDiffTrigger := githubissuedeliveryv1.DeriveDeliveryID(subID1, trigger2)
	if id1 == idDiffTrigger {
		t.Errorf("expected different IDs for different triggers, got same: %s", id1)
	}

	idDiffSub := githubissuedeliveryv1.DeriveDeliveryID(subID2, trigger1)
	if id1 == idDiffSub {
		t.Errorf("expected different IDs for different subscriptions, got same: %s", id1)
	}
}

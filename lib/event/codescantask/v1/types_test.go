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
	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
)

func TestCodeScanTaskEvent_EventMetadataAndSerialization(t *testing.T) {
	t.Parallel()

	evt := codescantaskv1.CodeScanTaskEvent{
		VCSProvider:        "github",
		VCSInstallationID:  "inst-123",
		VCSRepositoryID:    "repo-456",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		CommitSHA:          "abc1234",
		Branch:             "main",
		IsDefaultBranch:    true,
		ModifiedFiles:      []string{"src/index.ts"},
	}

	if evt.Kind() != "CodeScanTaskEvent" {
		t.Errorf("Kind() = %s, want CodeScanTaskEvent", evt.Kind())
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

	if env.Kind != "CodeScanTaskEvent" || env.APIVersion != "v1" {
		t.Errorf("unexpected envelope headers: %+v", env)
	}

	var parsed codescantaskv1.CodeScanTaskEvent
	if err := json.Unmarshal(env.Data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal inner data failed: %v", err)
	}
	if parsed.CommitSHA != "abc1234" {
		t.Errorf("parsed.CommitSHA = %s, want abc1234", parsed.CommitSHA)
	}
}

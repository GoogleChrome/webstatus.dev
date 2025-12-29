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

package producer

import (
	"context"
	"errors"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type mockSearchLister struct {
	searches []workertypes.SearchJob
	err      error
}

func (m *mockSearchLister) ListAllSavedSearches(_ context.Context) ([]workertypes.SearchJob, error) {
	return m.searches, m.err
}

type mockCommandPublisher struct {
	commands []workertypes.RefreshSearchCommand
	err      error
}

func (m *mockCommandPublisher) PublishRefreshCommand(_ context.Context, cmd workertypes.RefreshSearchCommand) error {
	m.commands = append(m.commands, cmd)

	return m.err
}

func TestProcessBatchUpdate(t *testing.T) {
	tests := []struct {
		name           string
		searches       []workertypes.SearchJob
		listerErr      error
		pubErr         error
		expectPubCalls int
		wantErr        bool
		transient      bool
	}{
		{
			name: "success with searches",
			searches: []workertypes.SearchJob{
				{ID: "s1", Query: "q=1"},
				{ID: "s2", Query: "q=2"},
			},
			listerErr:      nil,
			pubErr:         nil,
			expectPubCalls: 2,
			wantErr:        false,
			transient:      false,
		},
		{
			name:           "success empty list",
			searches:       []workertypes.SearchJob{},
			expectPubCalls: 0,
			listerErr:      nil,
			pubErr:         nil,
			wantErr:        false,
			transient:      false,
		},
		{
			name:           "lister error",
			listerErr:      errors.New("db error"),
			wantErr:        true,
			transient:      true,
			searches:       nil,
			pubErr:         nil,
			expectPubCalls: 0,
		},
		{
			name:           "publisher error",
			listerErr:      nil,
			searches:       []workertypes.SearchJob{{ID: "s1", Query: "q=1"}},
			pubErr:         errors.New("pub error"),
			wantErr:        true,
			transient:      true,
			expectPubCalls: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lister := &mockSearchLister{searches: tc.searches, err: tc.listerErr}
			pub := &mockCommandPublisher{err: tc.pubErr, commands: nil}
			handler := NewBatchUpdateHandler(lister, pub)

			err := handler.ProcessBatchUpdate(context.Background(), "trigger-1", workertypes.FrequencyImmediate)

			if (err != nil) != tc.wantErr {
				t.Errorf("ProcessBatchUpdate() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr && tc.transient {
				if !errors.Is(err, event.ErrTransientFailure) {
					t.Errorf("Expected error to be transient")
				}
			}

			if len(pub.commands) != tc.expectPubCalls {
				t.Errorf("Expected %d publish calls, got %d", tc.expectPubCalls, len(pub.commands))
			}

			// Verify command content for success case
			if !tc.wantErr && len(tc.searches) > 0 {
				if pub.commands[0].SearchID != "s1" {
					t.Errorf("Command data mismatch")
				}
				if pub.commands[0].Frequency != workertypes.FrequencyImmediate {
					t.Errorf("Frequency mismatch")
				}
			}
		})
	}
}

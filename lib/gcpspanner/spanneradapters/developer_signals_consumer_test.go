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

package spanneradapters

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/developersignaltypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/google/go-cmp/cmp"
)

type MockDeveloperSignalsClient struct {
	callHistory   []gcpspanner.FeatureDeveloperSignal
	shouldFail    bool
	failWithError error
}

func (m *MockDeveloperSignalsClient) SyncLatestFeatureDeveloperSignals(
	_ context.Context,
	data []gcpspanner.FeatureDeveloperSignal,
) error {
	m.callHistory = append(m.callHistory, data...)
	if m.shouldFail {
		return m.failWithError
	}

	return nil
}

func TestSyncLatestFeatureDeveloperSignals(t *testing.T) {
	testCases := []struct {
		name          string
		input         *developersignaltypes.FeatureDeveloperSignals
		mockClient    *MockDeveloperSignalsClient
		expectedError error
		expectedCalls []gcpspanner.FeatureDeveloperSignal
	}{
		{
			name: "Success",
			input: &developersignaltypes.FeatureDeveloperSignals{
				"feature1": {Upvotes: 100, Link: "link1"},
				"feature2": {Upvotes: 200, Link: "link2"},
			},
			mockClient:    &MockDeveloperSignalsClient{callHistory: nil, shouldFail: false, failWithError: nil},
			expectedError: nil,
			expectedCalls: []gcpspanner.FeatureDeveloperSignal{
				{WebFeatureKey: "feature1", Upvotes: 100, Link: "link1"},
				{WebFeatureKey: "feature2", Upvotes: 200, Link: "link2"},
			},
		},
		{
			name:          "Empty input",
			input:         &developersignaltypes.FeatureDeveloperSignals{},
			mockClient:    &MockDeveloperSignalsClient{callHistory: nil, shouldFail: false, failWithError: nil},
			expectedError: nil,
			expectedCalls: nil,
		},
		{
			name: "Spanner client error",
			input: &developersignaltypes.FeatureDeveloperSignals{
				"feature1": {Upvotes: 100, Link: "link1"},
			},
			mockClient: &MockDeveloperSignalsClient{
				shouldFail:    true,
				failWithError: errors.New("spanner error"),
				callHistory:   nil,
			},
			expectedError: errors.New("spanner error"),
			expectedCalls: []gcpspanner.FeatureDeveloperSignal{
				{WebFeatureKey: "feature1", Upvotes: 100, Link: "link1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			consumer := NewDeveloperSignalsConsumer(tc.mockClient)
			err := consumer.SyncLatestFeatureDeveloperSignals(context.Background(), tc.input)

			if !errors.Is(err, tc.expectedError) {
				if err.Error() != tc.expectedError.Error() {
					t.Errorf("unexpected error. got %v, want %v", err, tc.expectedError)
				}
			}

			// Sort slices for deterministic comparison
			cmpFunc := func(i, j gcpspanner.FeatureDeveloperSignal) int {
				if i.WebFeatureKey < j.WebFeatureKey {
					return -1
				}
				if i.WebFeatureKey > j.WebFeatureKey {
					return 1
				}

				return 0
			}
			slices.SortFunc(tc.mockClient.callHistory, cmpFunc)
			slices.SortFunc(tc.expectedCalls, cmpFunc)

			if diff := cmp.Diff(tc.expectedCalls, tc.mockClient.callHistory); diff != "" {
				t.Errorf("unexpected call history (-want +got): %s", diff)
			}
		})
	}
}

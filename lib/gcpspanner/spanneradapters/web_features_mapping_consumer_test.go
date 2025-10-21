// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"reflect"
	"sort"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
)

// mockWebFeaturesMappingClient is a mock implementation of the WebFeaturesMappingClient interface.
type mockWebFeaturesMappingClient struct {
	received []gcpspanner.WebFeaturesMappingData
}

func (m *mockWebFeaturesMappingClient) SyncWebFeaturesMappingData(
	_ context.Context, data []gcpspanner.WebFeaturesMappingData) error {
	m.received = data

	return nil
}

func TestSyncWebFeaturesMappingData(t *testing.T) {
	// Create a mock client
	mockClient := &mockWebFeaturesMappingClient{
		received: nil,
	}

	// Create the adapter with the mock client
	adapter := NewWebFeaturesMappingConsumer(mockClient)

	// Define test cases
	testCases := []struct {
		name        string
		input       webfeaturesmappingtypes.WebFeaturesMappings
		expected    []gcpspanner.WebFeaturesMappingData
		expectedErr error
	}{
		{
			name: "success",
			input: webfeaturesmappingtypes.WebFeaturesMappings{
				"feature1": {
					StandardsPositions: []webfeaturesmappingtypes.StandardsPosition{
						{
							Vendor:   "vendor1",
							Position: "position1",
							URL:      "url1",
							Concerns: nil,
						},
					},
				},
			},
			expected: []gcpspanner.WebFeaturesMappingData{
				{
					WebFeatureID: "feature1",
					VendorPositions: spanner.NullJSON{
						Value: `[{"vendor":"vendor1","position":"position1","url":"url1"}]`,
						Valid: true,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name:        "empty input",
			input:       webfeaturesmappingtypes.WebFeaturesMappings{},
			expected:    []gcpspanner.WebFeaturesMappingData{},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the mock client's received data
			mockClient.received = nil

			// Call the method
			err := adapter.SyncWebFeaturesMappingData(context.Background(), tc.input)

			// Check for errors
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedErr)
			}

			// Check if the received data matches the expected data
			if len(mockClient.received) != len(tc.expected) {
				t.Errorf("unexpected number of items. got %d, want %d", len(mockClient.received), len(tc.expected))
			}
			sort.Slice(mockClient.received, func(i, j int) bool {
				return mockClient.received[i].WebFeatureID < mockClient.received[j].WebFeatureID
			})
			sort.Slice(tc.expected, func(i, j int) bool {
				return tc.expected[i].WebFeatureID < tc.expected[j].WebFeatureID
			})
			if !reflect.DeepEqual(mockClient.received, tc.expected) {
				t.Errorf("unexpected data. got %+v, want %+v", mockClient.received, tc.expected)
			}
		})
	}
}

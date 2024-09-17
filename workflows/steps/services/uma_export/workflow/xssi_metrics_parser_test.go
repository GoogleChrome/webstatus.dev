// Copyright 2024 Google LLC
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

package workflow

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

func TestXSSIMetricsParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		inputData   string
		expected    metricdatatypes.BucketDataMetrics
		expectedErr error
	}{
		{
			name: "Valid JSON with XSSI prefix",
			inputData: `)]}\
{
	"r": {
		"123":
			{
				"rate": 0.5,
				"milestone": "89",
				"low_volume": true
			},
		"456":
			{
				"rate": 0.2,
				"milestone": "99",
				"low_volume": false
			}
	}
}`,
			expected: metricdatatypes.BucketDataMetrics{
				123: {Rate: 0.5, Milestone: "89", LowVolume: true},
				456: {Rate: 0.2, Milestone: "99", LowVolume: false},
			},
			expectedErr: nil,
		},
		{
			name: "Invalid JSON",
			inputData: `)]}\
{"invalid": }`,
			expected:    nil,
			expectedErr: errInvalidJSON,
		},
		{
			name: "Non-numeric bucket ID",
			inputData: `)]}\
{
	"r": {
		"abc":
			{
				"rate": 0.5,
				"milestone": "89",
				"low_volume": true
			},
		"456":
			{
				"rate": 0.2,
				"milestone": "99",
				"low_volume": false
			}
	}
}`,
			expected:    nil,
			expectedErr: errUnexpectedBucketID,
		},
		{
			name:        "Empty input",
			inputData:   "",
			expected:    nil,
			expectedErr: errMissingXSSIPrefix,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := XSSIMetricsParser{}
			ctx := context.Background()

			data := io.NopCloser(bytes.NewReader([]byte(tt.inputData)))
			got, err := parser.Parse(ctx, data)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				} else if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error '%v', but got '%v'", tt.expectedErr, err)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Expected %v, but got %v", tt.expected, got)
			}
		})
	}
}

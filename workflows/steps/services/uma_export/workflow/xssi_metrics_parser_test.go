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
{"abc": {"rate": 0.3}}`,
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
			ctx := context.Background() // Or create a context with values if needed

			data := io.NopCloser(bytes.NewReader([]byte(tt.inputData)))
			got, err := parser.Parse(ctx, data)

			// nolint:nestif // WONTFIX
			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				} else if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error '%v', but got '%v'", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Expected %v, but got %v", tt.expected, got)
				}
			}
		})
	}
}

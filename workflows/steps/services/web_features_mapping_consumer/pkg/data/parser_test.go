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

package data

import (
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name               string
		path               string
		expectedToExist    []string
		featureIDToCompare string
		expectedFeature    webfeaturesmappingtypes.FeatureMapping
		expectedError      error
	}{
		{
			name:               "valid json",
			path:               path.Join("testdata", "combined-data.json"),
			expectedToExist:    []string{"a", "abbr", "aborting"},
			featureIDToCompare: "compute-pressure",
			expectedFeature: webfeaturesmappingtypes.FeatureMapping{
				StandardsPositions: []webfeaturesmappingtypes.StandardsPosition{
					{
						Vendor:   "mozilla",
						Position: "",
						URL:      "https://github.com/mozilla/standards-positions/issues/521",
						Concerns: []string{},
					},
					{
						Vendor:   "webkit",
						Position: "oppose",
						URL:      "https://github.com/WebKit/standards-positions/issues/255",
						Concerns: []string{
							"privacy",
							"device independence",
						},
					},
				},
			},
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("unable to read file err %s", err.Error())
			}
			result, err := Parser{}.Parse(file)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error %v, got %v", tc.expectedError, err)
			}
			for _, key := range tc.expectedToExist {
				if _, ok := result[key]; !ok {
					t.Errorf("expected key %s to exist", key)
				}
			}
			if diff := cmp.Diff(tc.expectedFeature, result[tc.featureIDToCompare]); diff != "" {
				t.Errorf("unexpected result. (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParser_Error(t *testing.T) {
	testCases := []struct {
		name          string
		input         io.ReadCloser
		expectedError error
	}{
		{
			name:          "invalid json",
			input:         io.NopCloser(strings.NewReader("invalid")),
			expectedError: ErrUnexpectedFormat,
		},
		{
			name:          "empty input",
			input:         io.NopCloser(strings.NewReader("")),
			expectedError: ErrUnexpectedFormat,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{}
			_, err := p.Parse(tc.input)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}

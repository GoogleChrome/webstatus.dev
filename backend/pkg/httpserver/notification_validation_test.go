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

package httpserver

import (
	"errors"
	"testing"
)

func TestValidateSlackWebhookURL(t *testing.T) {
	testCases := []struct {
		name          string
		url           string
		expectedError error
	}{
		{
			name:          "Valid URL",
			url:           "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			expectedError: nil,
		},
		{
			name:          "Invalid Scheme (http)",
			url:           "http://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			expectedError: errInvalidSlackWebhookURL,
		},
		{
			name:          "Invalid Host",
			url:           "https://slack.hooks.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			expectedError: errInvalidSlackWebhookURL,
		},
		{
			name:          "Invalid Path (missing /services/)",
			url:           "https://hooks.slack.com/foo/bar",
			expectedError: errInvalidSlackWebhookURL,
		},
		{
			name:          "Invalid URL format",
			url:           "://foo-bar",
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSlackWebhookURL(tc.url)

			if tc.name == "Invalid URL format" {
				if err == nil {
					t.Error("Expected error for invalid URL format, got nil")
				}

				return
			}

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Expected error %v, got %v", tc.expectedError, err)
			}
		})
	}
}

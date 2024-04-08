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

package data

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
)

// nolint: gochecknoglobals // used for updating goldens
var (
	update = flag.Bool("update", false, "update golden files before testing")
)

func TestMain(m *testing.M) {
	flag.Parse()

	os.Exit(m.Run())
}

// By default data.json is huge. Only extract the keys we need.
func updateGolden(t *testing.T, path, location string) {
	ctx := context.Background()
	// 1. Download the JSON file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		t.Fatalf("unable to build request %s", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code when download the file: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading JSON data: %s", err)

	}

	// 2. Unmarshal the original JSON
	var originalData interface{}
	err = json.Unmarshal(body, &originalData)
	if err != nil {
		t.Fatalf("Error unmarshaling original JSON: %s", err)
	}

	// 3. Extract the "browsers" key
	dataMap, ok := originalData.(map[string]interface{})
	if !ok {
		t.Fatal("Original JSON is not in the expected format")
	}

	browsersData, ok := dataMap["browsers"]
	if !ok {
		t.Fatal("The 'browsers' key was not found")
	}

	outputData := map[string]interface{}{
		"browsers": browsersData,
	}

	// 4. Write to the new JSON file
	newFile, err := os.Create(path)
	if err != nil {
		t.Fatalf("Error creating file: %s", err)
	}
	defer newFile.Close()

	encoder := json.NewEncoder(newFile)
	err = encoder.Encode(outputData)
	if err != nil {
		t.Fatalf("Error encoding and writing JSON: %s", err)
	}

	t.Logf("Updated golden file saved to: %s", path)
}

func TestParse(t *testing.T) {
	testCases := []struct {
		name     string
		location string // For regenerating goldens
		path     string
	}{
		{
			name:     "data.json",
			location: "https://github.com/mdn/browser-compat-data/releases/download/v5.5.19/data.json",
			path:     path.Join("testdata", "data.json"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if update != nil && *update {
				updateGolden(t, tc.path, tc.location)
			}
			file, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("unable to read file err %s", err.Error())
			}
			result, err := Parser{}.Parse(file)
			if err != nil {
				t.Errorf("unable to parse file err %s", err.Error())
			}

			if len(result.BrowserData.Browsers) == 0 {
				t.Error("unexpected empty map")
			}
		})
	}

}

func TestParseError(t *testing.T) {
	testCases := []struct {
		name          string
		input         io.ReadCloser
		expectedError error
	}{
		{
			name:          "bad format",
			input:         io.NopCloser(strings.NewReader("Hello, world!")),
			expectedError: ErrUnexpectedFormat,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Parser{}.Parse(tc.input)
			if !errors.Is(err, tc.expectedError) {
				t.Errorf("unexpected error expected %v received %v", tc.expectedError, err)
			}
			if result != nil {
				t.Error("unexpected map")
			}
		})
	}
}

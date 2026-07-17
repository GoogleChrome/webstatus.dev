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
	"bytes"
	"testing"
)

func TestNewRSSRenderer(t *testing.T) {
	renderer := NewRSSRenderer()
	if renderer == nil {
		t.Fatal("NewRSSRenderer returned nil")
	}
	if renderer.tmpl == nil {
		t.Fatal("NewRSSRenderer returned renderer with nil template")
	}
}

func TestRenderRSSDescription(t *testing.T) {
	renderer := NewRSSRenderer()

	testCases := []struct {
		name             string
		data             RSSItemData
		expectedContains []string
	}{
		{
			name: "Basic Summary with Added Feature",
			data: RSSItemData{
				SummaryText:         "1 new feature matched",
				Added:               []string{"Feature A"},
				Removed:             nil,
				Changed:             nil,
				Moved:               nil,
				Split:               nil,
				Deleted:             nil,
				QueryErrors:         nil,
				ResolvedQueryErrors: nil,
				Truncated:           false,
			},
			expectedContains: []string{
				"Feature A",
				"Features Added",
			},
		},
		{
			name: "Removed Feature",
			data: RSSItemData{
				SummaryText:         "1 feature removed",
				Added:               nil,
				Removed:             []string{"Feature B"},
				Changed:             nil,
				Moved:               nil,
				Split:               nil,
				Deleted:             nil,
				QueryErrors:         nil,
				ResolvedQueryErrors: nil,
				Truncated:           false,
			},
			expectedContains: []string{
				"Feature B",
				"Features Removed",
			},
		},
		{
			name: "Changed Update",
			data: RSSItemData{
				SummaryText:         "1 feature updated",
				Added:               nil,
				Removed:             nil,
				Changed:             []string{"Feature C"},
				Moved:               nil,
				Split:               nil,
				Deleted:             nil,
				QueryErrors:         nil,
				ResolvedQueryErrors: nil,
				Truncated:           false,
			},
			expectedContains: []string{
				"Feature C",
				"Features Changed",
			},
		},
		{
			name: "HTML Escaping in Feature Name",
			data: RSSItemData{
				SummaryText:         "HTML escaping test",
				Added:               []string{"<link rel=\"dns-prefetch\">"},
				Removed:             nil,
				Changed:             nil,
				Moved:               nil,
				Split:               nil,
				Deleted:             nil,
				QueryErrors:         nil,
				ResolvedQueryErrors: nil,
				Truncated:           false,
			},
			expectedContains: []string{
				"&lt;link",
				"dns-prefetch",
			},
		},
		{
			name: "Truncated Summary",
			data: RSSItemData{
				SummaryText:         "Summary text",
				Added:               nil,
				Removed:             nil,
				Changed:             nil,
				Moved:               nil,
				Split:               nil,
				Deleted:             nil,
				QueryErrors:         nil,
				ResolvedQueryErrors: nil,
				Truncated:           true,
			},
			expectedContains: []string{
				"This summary has been truncated",
			},
		},
		{
			name: "Query Error Banner",
			data: RSSItemData{
				SummaryText:         "Query failure",
				Added:               nil,
				Removed:             nil,
				Changed:             nil,
				Moved:               nil,
				Split:               nil,
				Deleted:             nil,
				QueryErrors:         []string{"Invalid query grammar"},
				ResolvedQueryErrors: nil,
				Truncated:           false,
			},
			expectedContains: []string{
				"Query Errors",
				"Invalid query grammar",
			},
		},
		{
			name: "Resolved Query Error Banner",
			data: RSSItemData{
				SummaryText:         "Query recovered",
				Added:               nil,
				Removed:             nil,
				Changed:             nil,
				Moved:               nil,
				Split:               nil,
				Deleted:             nil,
				QueryErrors:         nil,
				ResolvedQueryErrors: []string{"Invalid query grammar"},
				Truncated:           false,
			},
			expectedContains: []string{
				"Query Recovered",
				"Tracking resumed cleanly from your baseline. Resolved issue: Invalid query grammar", //nolint:lll // WONTFIX - test string
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := renderer.RenderRSSDescription(tc.data)
			if err != nil {
				t.Fatalf("RenderRSSDescription failed: %v", err)
			}

			for _, expected := range tc.expectedContains {
				if !bytes.Contains([]byte(output), []byte(expected)) {
					t.Errorf("Expected output to contain %q, but it did not. Output: %s", expected, output)
				}
			}
		})
	}
}

func TestRenderRSSDescription_AllCategories(t *testing.T) {
	data := RSSItemData{
		SummaryText:         "Full feature summary",
		Added:               []string{"Added Feature"},
		Removed:             []string{"Removed Feature"},
		Changed:             []string{"Changed Feature"},
		Moved:               []string{"Moved Feature"},
		Split:               []string{"Split Feature"},
		Deleted:             []string{"Deleted Feature"},
		QueryErrors:         []string{"Saved search not found"},
		ResolvedQueryErrors: []string{"Invalid query grammar"},
		Truncated:           true,
	}

	renderer := NewRSSRenderer()
	output, err := renderer.RenderRSSDescription(data)
	if err != nil {
		t.Fatalf("RenderRSSDescription failed: %v", err)
	}

	expectedSections := []string{
		"Query Recovered",
		"Query Errors",
		"Features Added",
		"Features Removed",
		"Features Changed",
		"Features Moved/Renamed",
		"Features Split",
		"Features Deleted",
		"Note: This summary has been truncated.",
	}

	for _, section := range expectedSections {
		if !bytes.Contains([]byte(output), []byte(section)) {
			t.Errorf("expected RSS description to contain %q, got: %s", section, output)
		}
	}
}

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

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
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
		summary          workertypes.EventSummary
		expectedContains []string
	}{
		{
			name: "Basic Summary with Added Feature",
			summary: workertypes.EventSummary{
				SchemaVersion: workertypes.VersionEventSummaryV1,
				Text:          "1 new feature matched",
				Categories: workertypes.SummaryCategories{
					QueryChanged:    0,
					Added:           0,
					Removed:         0,
					Deleted:         0,
					Moved:           0,
					Split:           0,
					Updated:         0,
					UpdatedImpl:     0,
					UpdatedRename:   0,
					UpdatedBaseline: 0,
				},
				Truncated:      false,
				SnapshotOrigin: "",
				QueryErrors:    nil,
				Highlights: []workertypes.SummaryHighlight{
					{
						Type:           workertypes.SummaryHighlightTypeAdded,
						FeatureID:      "feature-a",
						FeatureName:    "Feature A",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: nil,
						Moved:          nil,
						Split:          nil,
					},
				},
			},
			expectedContains: []string{
				"Feature A",
				"Features Added",
			},
		},
		{
			name: "Removed Feature",
			summary: workertypes.EventSummary{
				SchemaVersion: workertypes.VersionEventSummaryV1,
				Text:          "1 feature removed",
				Categories: workertypes.SummaryCategories{
					QueryChanged:    0,
					Added:           0,
					Removed:         0,
					Deleted:         0,
					Moved:           0,
					Split:           0,
					Updated:         0,
					UpdatedImpl:     0,
					UpdatedRename:   0,
					UpdatedBaseline: 0,
				},
				Truncated:      false,
				SnapshotOrigin: "",
				QueryErrors:    nil,
				Highlights: []workertypes.SummaryHighlight{
					{
						Type:           workertypes.SummaryHighlightTypeRemoved,
						FeatureID:      "feature-b",
						FeatureName:    "Feature B",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: nil,
						Moved:          nil,
						Split:          nil,
					},
				},
			},
			expectedContains: []string{
				"Feature B",
				"Features Removed",
			},
		},
		{
			name: "Other Update",
			summary: workertypes.EventSummary{
				SchemaVersion: workertypes.VersionEventSummaryV1,
				Text:          "1 feature updated",
				Categories: workertypes.SummaryCategories{
					QueryChanged:    0,
					Added:           0,
					Removed:         0,
					Deleted:         0,
					Moved:           0,
					Split:           0,
					Updated:         0,
					UpdatedImpl:     0,
					UpdatedRename:   0,
					UpdatedBaseline: 0,
				},
				Truncated:      false,
				SnapshotOrigin: "",
				QueryErrors:    nil,
				Highlights: []workertypes.SummaryHighlight{
					{
						Type:           workertypes.SummaryHighlightTypeChanged,
						FeatureID:      "feature-c",
						FeatureName:    "Feature C",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: nil,
						Moved:          nil,
						Split:          nil,
					},
				},
			},
			expectedContains: []string{
				"Feature C",
				"Other Updates",
			},
		},
		{
			name: "HTML Escaping in Feature Name",
			summary: workertypes.EventSummary{
				SchemaVersion: workertypes.VersionEventSummaryV1,
				Text:          "HTML escaping test",
				Categories: workertypes.SummaryCategories{
					QueryChanged:    0,
					Added:           0,
					Removed:         0,
					Deleted:         0,
					Moved:           0,
					Split:           0,
					Updated:         0,
					UpdatedImpl:     0,
					UpdatedRename:   0,
					UpdatedBaseline: 0,
				},
				Truncated:      false,
				SnapshotOrigin: "",
				QueryErrors:    nil,
				Highlights: []workertypes.SummaryHighlight{
					{
						Type:           workertypes.SummaryHighlightTypeAdded,
						FeatureID:      "feature-html",
						FeatureName:    "<link rel=\"dns-prefetch\">",
						Docs:           nil,
						NameChange:     nil,
						BaselineChange: nil,
						BrowserChanges: nil,
						Moved:          nil,
						Split:          nil,
					},
				},
			},
			expectedContains: []string{
				"&lt;link",
				"dns-prefetch",
			},
		},
		{
			name: "Truncated Summary",
			summary: workertypes.EventSummary{
				SchemaVersion: workertypes.VersionEventSummaryV1,
				Text:          "Summary text",
				Categories: workertypes.SummaryCategories{
					QueryChanged:    0,
					Added:           0,
					Removed:         0,
					Deleted:         0,
					Moved:           0,
					Split:           0,
					Updated:         0,
					UpdatedImpl:     0,
					UpdatedRename:   0,
					UpdatedBaseline: 0,
				},
				Truncated:      true,
				SnapshotOrigin: "",
				QueryErrors:    nil,
				Highlights:     []workertypes.SummaryHighlight{},
			},
			expectedContains: []string{
				"This summary has been truncated",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := renderer.RenderRSSDescription(tc.summary)
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

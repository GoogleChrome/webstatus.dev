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
	"encoding/json"
	"errors"
	"testing"
	"time"

	featurelistv1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelist/v1"
	featurelistdiffv1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelistdiff/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
	"github.com/GoogleChrome/webstatus.dev/workers/event_producer/pkg/differ"
	"github.com/google/go-cmp/cmp"
)

// A concrete type for the generic parameter S in our tests.
type testSnapshot struct {
	Data   string `json:"data"`
	IDVal  string `json:"idVal"`
	idFunc func() string
}

// mock ID method for snapshot interface.
func (s testSnapshot) ID() string {
	if s.idFunc != nil {
		return s.idFunc()
	}

	return s.IDVal
}

// TestGenericStateAdapter_Load tests the loading logic of the generic adapter.
func TestGenericStateAdapter_Load(t *testing.T) {
	testErr := errors.New("test error")
	testSnapshotMap := map[string]comparables.Feature{"feat-a": {
		ID:             "feat-a",
		Name:           generic.UnsetOpt[string](),
		BaselineStatus: generic.UnsetOpt[comparables.BaselineState](),
		BrowserImpls:   generic.UnsetOpt[comparables.BrowserImplementations](),
		Docs:           generic.UnsetOpt[comparables.Docs](),
	}}

	tests := []struct {
		name          string
		inputBytes    []byte
		mockMigrator  migratorFunc
		mockConverter stateConverter[testSnapshot]
		wantSnapshot  map[string]comparables.Feature
		wantID        string
		wantSignature string
		wantIsEmpty   bool
		wantErr       error
	}{
		{
			name:         "Empty input bytes",
			inputBytes:   nil,
			mockMigrator: func(b []byte) ([]byte, error) { return b, nil },
			mockConverter: func(_ *testSnapshot) (map[string]comparables.Feature, string) {
				return nil, ""
			},
			wantSnapshot:  nil,
			wantID:        "",
			wantSignature: "",
			wantIsEmpty:   true,
			wantErr:       nil,
		},
		{
			name:         "Successful load",
			inputBytes:   []byte(`{"data":"some-data", "idVal": "state-123"}`),
			mockMigrator: func(b []byte) ([]byte, error) { return b, nil },
			mockConverter: func(s *testSnapshot) (map[string]comparables.Feature, string) {
				if s.Data != "some-data" {
					t.Errorf("converter received unexpected data: %s", s.Data)
				}

				return testSnapshotMap, "sig-123"
			},
			wantSnapshot:  testSnapshotMap,
			wantID:        "state-123",
			wantSignature: "sig-123",
			wantIsEmpty:   false,
			wantErr:       nil,
		},
		{
			name:         "Migrator fails",
			inputBytes:   []byte("data"),
			mockMigrator: func(_ []byte) ([]byte, error) { return nil, testErr },
			mockConverter: func(_ *testSnapshot) (map[string]comparables.Feature, string) {
				t.Error("converter should not be called when migrator fails")

				return nil, ""
			},
			wantSnapshot:  nil,
			wantID:        "",
			wantSignature: "",
			wantIsEmpty:   false,
			wantErr:       testErr,
		},
		{
			name:         "Unmarshal fails",
			inputBytes:   []byte("invalid-json"),
			mockMigrator: func(b []byte) ([]byte, error) { return b, nil },
			mockConverter: func(_ *testSnapshot) (map[string]comparables.Feature, string) {
				t.Error("converter should not be called when unmarshal fails")

				return nil, ""
			},
			wantSnapshot:  nil,
			wantID:        "",
			wantSignature: "",
			wantIsEmpty:   false,
			wantErr:       ErrInvalidFormat,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adapter := newGenericStateAdapter(tc.mockMigrator, tc.mockConverter, nil)

			gotSnapshot, gotID, gotSignature, gotIsEmpty, err := adapter.Load(tc.inputBytes)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Load() error = %v, want type/is %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tc.wantSnapshot, gotSnapshot); diff != "" {
				t.Errorf("Load() snapshot mismatch (-want +got):\n%s", diff)
			}
			if gotID != tc.wantID {
				t.Errorf("Load() id mismatch: got %q, want %q", gotID, tc.wantID)
			}
			if gotSignature != tc.wantSignature {
				t.Errorf("Load() signature mismatch: got %q, want %q", gotSignature, tc.wantSignature)
			}
			if gotIsEmpty != tc.wantIsEmpty {
				t.Errorf("Load() isEmpty mismatch: got %v, want %v", gotIsEmpty, tc.wantIsEmpty)
			}
		})
	}
}

// TestV1DiffSerializer_Serialize tests the V1 diff serializer.
func TestV1DiffSerializer_Serialize(t *testing.T) {
	serializer := NewV1DiffSerializer()
	diff := &featurelistdiffv1.FeatureDiff{
		QueryChanged: false,
		Added:        []featurelistdiffv1.FeatureAdded{{ID: "feat-a", Name: "Feature A", Reason: "", Docs: nil}},
		Deleted:      nil,
		Removed:      nil,
		Modified:     nil,
		Moves:        nil,
		Splits:       nil,
	}
	metadata := differ.DiffMetadata{
		ID:              "diff-id1",
		EventID:         "event-1",
		SearchID:        "search-1",
		NewStateID:      "state-2",
		PreviousStateID: "state-1",
	}
	now := time.Now()

	bytes, err := serializer.Serialize(metadata.ID, metadata.SearchID,
		metadata.EventID, metadata.NewStateID, metadata.PreviousStateID, diff, now)
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if kind, _ := raw["kind"].(string); kind != featurelistdiffv1.KindFeatureListDiff {
		t.Errorf("envelope.Kind mismatch: got %q, want %q", kind, featurelistdiffv1.KindFeatureListDiff)
	}
	if version, _ := raw["apiVersion"].(string); version != featurelistdiffv1.V1FeatureListDiff {
		t.Errorf("envelope.Version mismatch: got %q, want %q", version, featurelistdiffv1.V1FeatureListDiff)
	}

	// Remarshal the inner part to test the payload
	payloadBytes, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot featurelistdiffv1.FeatureListDiffSnapshot
	if err := json.Unmarshal(payloadBytes, &snapshot); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if snapshot.Metadata.EventID != metadata.EventID {
		t.Errorf("metadata.EventID mismatch")
	}
	if len(snapshot.Data.Added) != 1 {
		t.Errorf("snapshot.Data was not serialized correctly")
	}
}

// TestConversionFunctions tests the round trip of V1 <-> Comparable conversions.
func TestConversionFunctions(t *testing.T) {
	now := time.Now()
	versionStr := "1.0"

	// Define a fully populated canonical feature
	canonicalFeature := comparables.Feature{
		ID:   "feat-1",
		Name: generic.OptionallySet[string]{Value: "Feature One", IsSet: true},
		BaselineStatus: generic.OptionallySet[comparables.BaselineState]{
			IsSet: true,
			Value: comparables.BaselineState{
				Status:   generic.OptionallySet[backend.BaselineInfoStatus]{Value: backend.Widely, IsSet: true},
				LowDate:  generic.OptionallySet[*time.Time]{Value: &now, IsSet: true},
				HighDate: generic.UnsetOpt[*time.Time](),
			},
		},
		BrowserImpls: generic.OptionallySet[comparables.BrowserImplementations]{
			IsSet: true,
			Value: comparables.BrowserImplementations{
				Chrome: generic.OptionallySet[comparables.BrowserState]{
					IsSet: true,
					Value: comparables.BrowserState{
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Available, IsSet: true},
						Version: generic.OptionallySet[*string]{Value: &versionStr, IsSet: true},
						Date:    generic.OptionallySet[*time.Time]{Value: &now, IsSet: true},
					},
				},
				ChromeAndroid: generic.UnsetOpt[comparables.BrowserState](),
				Edge:          generic.UnsetOpt[comparables.BrowserState](),
				Firefox: generic.OptionallySet[comparables.BrowserState]{
					IsSet: true,
					Value: comparables.BrowserState{
						Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: backend.Unavailable, IsSet: true},
						Version: generic.UnsetOpt[*string](),
						Date:    generic.UnsetOpt[*time.Time](),
					},
				},
				FirefoxAndroid: generic.UnsetOpt[comparables.BrowserState](),
				Safari:         generic.UnsetOpt[comparables.BrowserState](), // Unset browser
				SafariIos:      generic.UnsetOpt[comparables.BrowserState](),
			},
		},
		Docs: generic.UnsetOpt[comparables.Docs](),
	}

	// 1. Convert Canonical -> V1
	v1Feature := convertComparableToV1Feature(canonicalFeature)

	// Assert V1 structure is correct
	if !v1Feature.BrowserImpls.Value.Chrome.IsSet ||
		!v1Feature.BrowserImpls.Value.Chrome.Value.Status.IsSet ||
		v1Feature.BrowserImpls.Value.Chrome.Value.Status.Value != featurelistv1.Available {
		t.Error("failed to convert browser status to V1")
	}
	if v1Feature.BrowserImpls.Value.Safari.IsSet {
		t.Error("expected unset Safari to remain unset in V1")
	}
	if !v1Feature.BaselineStatus.IsSet ||
		v1Feature.BaselineStatus.Value.Status.Value != featurelistv1.Widely {
		t.Error("failed to convert baseline status to V1")
	}

	// 2. Convert V1 -> Canonical (Round trip)
	roundTrippedFeature := convertV1FeatureToComparable(v1Feature)

	// 3. Compare original with round-tripped
	if diff := cmp.Diff(canonicalFeature, roundTrippedFeature); diff != "" {
		t.Errorf("round trip conversion mismatch (-want +got):\n%s", diff)
	}
}

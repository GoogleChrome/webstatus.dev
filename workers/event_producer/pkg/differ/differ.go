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

package differ

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	v1 "github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurediff/v1"
	"github.com/google/uuid"
)

// idGenerator abstracts the creation of unique identifiers for states and events.
type idGenerator interface {
	NewStateID() string
	NewDiffID() string
}

type defaultIDGenerator struct{}

func (g *defaultIDGenerator) NewStateID() string {
	return fmt.Sprintf("state_%s", g.newUUID())
}

func (g *defaultIDGenerator) NewDiffID() string {
	return fmt.Sprintf("diff_%s", g.newUUID())
}

func (g *defaultIDGenerator) newUUID() string {
	return uuid.New().String()
}

// Run executes the core diffing pipeline.
func (d *FeatureDiffer) Run(ctx context.Context, searchID string, query string,
	eventID string,
	previousStateBytes []byte) (*DiffResult, error) {
	result := new(DiffResult)
	result.Format = BlobFormatJSON
	result.DiffID = d.idGen.NewDiffID()
	result.StateID = d.idGen.NewStateID()

	// 1. Load Context
	prevCtx, err := d.loadPreviousContext(previousStateBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load previous state: %w", ErrFatal, err)
	}

	// 2. Plan
	plan := d.determinePlan(query, prevCtx)

	// 3. Execute Fetch
	data, err := d.executePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch data: %w", ErrTransient, err)
	}
	data.OldSnapshot = prevCtx.Snapshot

	// 4. Compute Pure Diff
	var diff *v1.LatestFeatureDiff
	// We check data.TargetSnapshot != nil because if the Flush Strategy failed (in executePlan),
	// it returns nil to signal "Skip Diffing".
	// toSnapshot() guarantees a non-nil map (empty map) for valid empty results,
	// so nil strictly means "Data Not Available".
	if !plan.IsColdStart && data.TargetSnapshot != nil {
		diff = calculateDiff(data.OldSnapshot, data.TargetSnapshot)
	} else {
		diff = new(v1.LatestFeatureDiff)
	}

	// 5. Reconcile History
	if len(diff.Removed) > 0 && !plan.IsColdStart {
		diff, err = d.reconcileHistory(ctx, diff)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to reconcile history: %w", ErrTransient, err)
		}
	}

	if plan.QueryChanged {
		diff.QueryChanged = true
	}

	// 6. Finalize (Sort & Decide)
	if diff != nil {
		diff.Sort()
	}

	// 7. Output Decision
	result.HasChanges = diff.HasChanges() || plan.IsColdStart || plan.QueryChanged
	if !result.HasChanges {
		return result, nil
	}

	// Serialize State
	result.StateBytes, err = d.serializeState(searchID, result.StateID, query, eventID, data.NewSnapshot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize new state: %w", ErrFatal, err)
	}

	// Calculate DB Metadata (Reasons & Summary)
	result.Reasons = determineReasons(diff)
	result.Summary = diff.Summarize()

	// Serialize Diff (Lineage Tracking)
	result.DiffBytes, err = d.serializeDiff(searchID, eventID, result.DiffID, prevCtx.ID, result.StateID, diff)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize diff: %w", ErrFatal, err)
	}

	return result, nil
}

func (d *FeatureDiffer) serializeState(
	searchID, stateID, query, eventID string,
	snapshot map[string]ComparableFeature,
) ([]byte, error) {
	payload := FeatureListSnapshot{
		Metadata: StateMetadata{
			GeneratedAt:    d.now(),
			ID:             stateID,
			SearchID:       searchID,
			QuerySignature: query,
			EventID:        eventID,
		},
		Data: FeatureListData{
			Features: snapshot,
		},
	}

	return blobtypes.NewBlob(payload)
}

func (d *FeatureDiffer) serializeDiff(
	searchID, eventID, diffID, previousStateID, newStateID string,
	diff *v1.LatestFeatureDiff,
) ([]byte, error) {
	diffPayload := v1.LatestFeatureDiffSnapshot{
		Metadata: v1.DiffMetadataV1{
			GeneratedAt:     d.now(),
			EventID:         eventID,
			SearchID:        searchID,
			PreviousStateID: previousStateID,
			NewStateID:      newStateID,
			ID:              diffID,
		},
		Data: *diff,
	}

	return blobtypes.NewBlob(diffPayload)
}

// --- Internal Helper: Context Loading ---

type previousContext struct {
	ID        string
	Signature string
	Snapshot  map[string]ComparableFeature
	IsEmpty   bool
}

func (d *FeatureDiffer) loadPreviousContext(bytes []byte) (previousContext, error) {
	if len(bytes) == 0 {
		return previousContext{
			IsEmpty:   true,
			Signature: "",
			Snapshot:  nil,
			ID:        "",
		}, nil
	}

	migratedBytes, err := blobtypes.Apply[FeatureListSnapshot](d.migrator, bytes)
	if err != nil {
		return previousContext{}, err
	}

	var snapshot FeatureListSnapshot
	if err := json.Unmarshal(migratedBytes, &snapshot); err != nil {
		return previousContext{}, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return previousContext{
		ID:        snapshot.Metadata.ID,
		Signature: snapshot.Metadata.QuerySignature,
		Snapshot:  snapshot.Data.Features,
		IsEmpty:   false,
	}, nil
}

func determineReasons(diff *v1.LatestFeatureDiff) []string {
	var reasons []string
	if diff.QueryChanged {
		reasons = append(reasons, "QUERY_EDITED")
	}
	if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 ||
		len(diff.Moves) > 0 || len(diff.Splits) > 0 {
		reasons = append(reasons, "DATA_UPDATED")
	}

	return reasons
}

// --- Internal Helper: Planning ---

type executionPlan struct {
	IsColdStart   bool
	QueryChanged  bool
	CurrentQuery  string
	PreviousQuery string
}

func (d *FeatureDiffer) determinePlan(currentQuery string, prev previousContext) executionPlan {
	plan := executionPlan{
		CurrentQuery:  currentQuery,
		PreviousQuery: "",
		IsColdStart:   false,
		QueryChanged:  false,
	}

	if prev.IsEmpty {
		plan.IsColdStart = true

		return plan
	}

	if prev.Signature != currentQuery {
		plan.QueryChanged = true
		plan.PreviousQuery = prev.Signature
	}

	return plan
}

// --- Internal Helper: Execution ---

type executionData struct {
	OldSnapshot    map[string]ComparableFeature
	TargetSnapshot map[string]ComparableFeature
	NewSnapshot    map[string]ComparableFeature
}

func (d *FeatureDiffer) executePlan(ctx context.Context, plan executionPlan) (executionData, error) {
	data := executionData{
		OldSnapshot:    nil,
		TargetSnapshot: nil,
		NewSnapshot:    nil,
	}

	newLive, err := d.client.FetchFeatures(ctx, plan.CurrentQuery)
	if err != nil {
		return data, err
	}
	data.NewSnapshot = toSnapshot(newLive)

	if plan.IsColdStart {
		return data, nil
	}

	if plan.QueryChanged {
		oldLive, err := d.client.FetchFeatures(ctx, plan.PreviousQuery)
		if err == nil {
			data.TargetSnapshot = toSnapshot(oldLive)
		} else {
			// Fallback: If old query fails, we return nil TargetSnapshot.
			// Run() detects this and skips diffing, treating it as a silent reset.
			return executionData{
				NewSnapshot:    data.NewSnapshot,
				TargetSnapshot: nil,
				OldSnapshot:    nil}, nil
		}
	} else {
		data.TargetSnapshot = data.NewSnapshot
	}

	return data, nil
}

func toSnapshot(features []backend.Feature) map[string]ComparableFeature {
	m := make(map[string]ComparableFeature)
	for _, f := range features {
		m[f.FeatureId] = toComparable(f)
	}

	return m
}

func toComparable(f backend.Feature) ComparableFeature {
	status := backend.Limited
	var lowDate, highDate *time.Time
	if f.Baseline != nil {
		if f.Baseline.Status != nil {
			status = *f.Baseline.Status
		}
		if f.Baseline.LowDate != nil {
			t := f.Baseline.LowDate.Time
			lowDate = &t
		}
		if f.Baseline.HighDate != nil {
			t := f.Baseline.HighDate.Time
			highDate = &t
		}
	}

	baseline := BaselineState{
		Status:   OptionallySet[backend.BaselineInfoStatus]{Value: status, IsSet: true},
		LowDate:  OptionallySet[*time.Time]{Value: lowDate, IsSet: true},
		HighDate: OptionallySet[*time.Time]{Value: highDate, IsSet: true},
	}

	docs := Docs{
		MdnDocs: OptionallySet[[]MdnDoc]{
			Value: nil,
			// TODO: Set to true when https://github.com/GoogleChrome/webstatus.dev/issues/930 is supported.
			IsSet: false,
		},
	}

	cf := ComparableFeature{
		ID:             f.FeatureId,
		Name:           OptionallySet[string]{Value: f.Name, IsSet: true},
		BaselineStatus: OptionallySet[BaselineState]{Value: baseline, IsSet: true},
		Docs:           OptionallySet[Docs]{Value: docs, IsSet: true},
		BrowserImpls: OptionallySet[BrowserImplementations]{
			Value: BrowserImplementations{
				Chrome: OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: OptionallySet[*string]{Value: nil, IsSet: false},
				}},
				ChromeAndroid: OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: OptionallySet[*string]{Value: nil, IsSet: false},
				}},
				Edge: OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: OptionallySet[*string]{Value: nil, IsSet: false},
				}},
				Firefox: OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: OptionallySet[*string]{Value: nil, IsSet: false},
				}},
				FirefoxAndroid: OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: OptionallySet[*string]{Value: nil, IsSet: false},
				}},
				Safari: OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: OptionallySet[*string]{Value: nil, IsSet: false},
				}},
				SafariIos: OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: OptionallySet[*string]{Value: nil, IsSet: false},
				}},
			},
			IsSet: true,
		},
	}

	if f.BrowserImplementations != nil {
		raw := *f.BrowserImplementations
		getStatus := func(key string) OptionallySet[BrowserState] {
			if impl, ok := raw[key]; ok && impl.Status != nil {
				var t *time.Time
				if impl.Date != nil {
					val := impl.Date.Time
					t = &val
				}
				bs := BrowserState{
					Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: *impl.Status, IsSet: true},
					Date:    OptionallySet[*time.Time]{Value: t, IsSet: true},
					Version: OptionallySet[*string]{Value: impl.Version, IsSet: true},
				}

				return OptionallySet[BrowserState]{Value: bs, IsSet: true}
			}

			return OptionallySet[BrowserState]{IsSet: false, Value: BrowserState{
				Status:  OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
				Date:    OptionallySet[*time.Time]{Value: nil, IsSet: false},
				Version: OptionallySet[*string]{Value: nil, IsSet: false},
			}}
		}

		cf.BrowserImpls.Value.Chrome = getStatus(string(backend.Chrome))
		cf.BrowserImpls.Value.ChromeAndroid = getStatus(string(backend.ChromeAndroid))
		cf.BrowserImpls.Value.Edge = getStatus(string(backend.Edge))
		cf.BrowserImpls.Value.Firefox = getStatus(string(backend.Firefox))
		cf.BrowserImpls.Value.FirefoxAndroid = getStatus(string(backend.FirefoxAndroid))
		cf.BrowserImpls.Value.Safari = getStatus(string(backend.Safari))
		cf.BrowserImpls.Value.SafariIos = getStatus(string(backend.SafariIos))
	}

	return cf
}

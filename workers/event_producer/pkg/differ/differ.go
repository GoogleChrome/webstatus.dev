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
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	snapshotV1 "github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurelistsnapshot/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurestate"
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
	diffResult, err := d.comparator.Compare(data.OldSnapshot, data.TargetSnapshot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to compare snapshots: %w", ErrFatal, err)
	}

	// 5. Reconcile History
	reconciledDiff, err := d.comparator.ReconcileHistory(ctx, diffResult.Diff)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to reconcile history: %w", ErrTransient, err)
	}
	diffResult.Diff = reconciledDiff

	if plan.QueryChanged {
		diffResult.SetQueryChanged(true)
	}

	// 7. Output Decision
	result.HasChanges = diffResult.HasChanges() || plan.IsColdStart || plan.QueryChanged
	if !result.HasChanges {
		return result, nil
	}

	// Serialize State
	result.StateBytes, err = d.serializeState(searchID, result.StateID, query, eventID, data.NewSnapshot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize new state: %w", ErrFatal, err)
	}

	// Calculate DB Metadata (Reasons & Summary)
	result.Reasons = determineReasons(diffResult)
	result.Summary = diffResult.Summarize()

	// Serialize Diff
	result.DiffBytes, err = diffResult.Bytes()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize diff: %w", ErrFatal, err)
	}

	return result, nil
}

func (d *FeatureDiffer) serializeState(
	searchID, stateID, query, eventID string,
	snapshot map[string]featurestate.ComparableFeature,
) ([]byte, error) {
	payload := snapshotV1.FeatureListSnapshotV1{
		Metadata: snapshotV1.StateMetadataV1{
			GeneratedAt:    d.now(),
			ID:             stateID,
			SearchID:       searchID,
			QuerySignature: query,
			EventID:        eventID,
		},
		Data: snapshotV1.FeatureListDataV1{
			Features: snapshot,
		},
	}

	return blobtypes.NewBlob(payload)
}

// --- Internal Helper: Context Loading ---

type previousContext struct {
	ID        string
	Signature string
	Snapshot  map[string]featurestate.ComparableFeature
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

	migratedBytes, err := blobtypes.Apply[snapshotV1.FeatureListSnapshotV1](d.migrator, bytes)
	if err != nil {
		return previousContext{}, err
	}

	var snapshot snapshotV1.FeatureListSnapshotV1
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

func determineReasons(diff *workertypes.DiffResult) []string {
	var reasons []string
	summary := diff.Summarize()
	if summary.Categories.QueryChanged > 0 {
		reasons = append(reasons, "QUERY_EDITED")
	}
	if summary.Categories.Added > 0 || summary.Categories.Removed > 0 || summary.Categories.Updated > 0 ||
		summary.Categories.Moved > 0 || summary.Categories.Split > 0 {
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
	OldSnapshot    map[string]featurestate.ComparableFeature
	TargetSnapshot map[string]featurestate.ComparableFeature
	NewSnapshot    map[string]featurestate.ComparableFeature
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

func toSnapshot(features []backend.Feature) map[string]featurestate.ComparableFeature {
	m := make(map[string]featurestate.ComparableFeature)
	for _, f := range features {
		m[f.FeatureId] = toComparable(f)
	}

	return m
}

func toComparable(f backend.Feature) featurestate.ComparableFeature {
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

	baseline := featurestate.BaselineState{
		Status:   featurestate.OptionallySet[backend.BaselineInfoStatus]{Value: status, IsSet: true},
		LowDate:  featurestate.OptionallySet[*time.Time]{Value: lowDate, IsSet: true},
		HighDate: featurestate.OptionallySet[*time.Time]{Value: highDate, IsSet: true},
	}

	docs := featurestate.Docs{
		MdnDocs: featurestate.OptionallySet[[]featurestate.MdnDoc]{
			Value: nil,
			// TODO: Set to true when https://github.com/GoogleChrome/webstatus.dev/issues/930 is supported.
			IsSet: false,
		},
	}

	cf := featurestate.ComparableFeature{
		ID:             f.FeatureId,
		Name:           featurestate.OptionallySet[string]{Value: f.Name, IsSet: true},
		BaselineStatus: featurestate.OptionallySet[featurestate.BaselineState]{Value: baseline, IsSet: true},
		Docs:           featurestate.OptionallySet[featurestate.Docs]{Value: docs, IsSet: true},
		BrowserImpls: featurestate.OptionallySet[featurestate.BrowserImplementations]{
			Value: featurestate.BrowserImplementations{
				Chrome: featurestate.OptionallySet[featurestate.BrowserState]{
					Value: featurestate.BrowserState{
						Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
						Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
						Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
					}, IsSet: false},
				ChromeAndroid: featurestate.OptionallySet[featurestate.BrowserState]{
					Value: featurestate.BrowserState{
						Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
						Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
						Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
					}, IsSet: false},
				Edge: featurestate.OptionallySet[featurestate.BrowserState]{
					Value: featurestate.BrowserState{
						Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
						Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
						Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
					}, IsSet: false},
				Firefox: featurestate.OptionallySet[featurestate.BrowserState]{
					Value: featurestate.BrowserState{
						Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
						Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
						Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
					}, IsSet: false},
				FirefoxAndroid: featurestate.OptionallySet[featurestate.BrowserState]{
					Value: featurestate.BrowserState{
						Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
						Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
						Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
					}, IsSet: false},
				Safari: featurestate.OptionallySet[featurestate.BrowserState]{
					Value: featurestate.BrowserState{
						Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
						Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
						Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
					}, IsSet: false},
				SafariIos: featurestate.OptionallySet[featurestate.BrowserState]{
					Value: featurestate.BrowserState{
						Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
						Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
						Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
					}, IsSet: false},
			},
			IsSet: true,
		},
	}

	if f.BrowserImplementations != nil {
		raw := *f.BrowserImplementations
		getStatus := func(key string) featurestate.OptionallySet[featurestate.BrowserState] {
			if impl, ok := raw[key]; ok && impl.Status != nil {
				var t *time.Time
				if impl.Date != nil {
					val := impl.Date.Time
					t = &val
				}
				bs := featurestate.BrowserState{
					Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: *impl.Status, IsSet: true},
					Date:    featurestate.OptionallySet[*time.Time]{Value: t, IsSet: true},
					Version: featurestate.OptionallySet[*string]{Value: impl.Version, IsSet: true},
				}

				return featurestate.OptionallySet[featurestate.BrowserState]{Value: bs, IsSet: true}
			}

			return featurestate.OptionallySet[featurestate.BrowserState]{
				IsSet: false,
				Value: featurestate.BrowserState{
					Status:  featurestate.OptionallySet[backend.BrowserImplementationStatus]{Value: "", IsSet: false},
					Date:    featurestate.OptionallySet[*time.Time]{Value: nil, IsSet: false},
					Version: featurestate.OptionallySet[*string]{Value: nil, IsSet: false},
				}}
		}

		cf.BrowserImpls.Value.SetBrowserState(backend.Chrome, getStatus(string(backend.Chrome)))
		cf.BrowserImpls.Value.SetBrowserState(backend.ChromeAndroid, getStatus(string(backend.ChromeAndroid)))
		cf.BrowserImpls.Value.SetBrowserState(backend.Edge, getStatus(string(backend.Edge)))
		cf.BrowserImpls.Value.SetBrowserState(backend.Firefox, getStatus(string(backend.Firefox)))
		cf.BrowserImpls.Value.SetBrowserState(backend.FirefoxAndroid, getStatus(string(backend.FirefoxAndroid)))
		cf.BrowserImpls.Value.SetBrowserState(backend.Safari, getStatus(string(backend.Safari)))
		cf.BrowserImpls.Value.SetBrowserState(backend.SafariIos, getStatus(string(backend.SafariIos)))
	}

	return cf
}

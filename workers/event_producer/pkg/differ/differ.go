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
	"fmt"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
	"github.com/google/uuid"
)

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
func (d *FeatureDiffer[D]) Run(ctx context.Context, searchID string, query string, eventID string,
	previousStateBytes []byte) (*DiffResult, error) {
	// 1. Load Context
	snapshot, id, signature, isEmpty, err := d.stateAdapter.Load(previousStateBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load previous state: %w", ErrFatal, err)
	}
	prevCtx := previousContext{
		Signature: signature,
		Snapshot:  snapshot,
		IsEmpty:   isEmpty,
		ID:        id,
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
	// We check data.TargetSnapshot != nil because if the Flush Strategy failed (in executePlan),
	// it returns nil to signal "Skip Diffing".
	// toSnapshot() guarantees a non-nil map (empty map) for valid empty results,
	// so nil strictly means "Data Not Available".
	if !plan.IsColdStart && data.TargetSnapshot != nil {
		d.workflow.CalculateDiff(data.OldSnapshot, data.TargetSnapshot)
	}

	// 5. Reconcile History
	if d.workflow.HasRemovedFeatures() && !plan.IsColdStart {
		err = d.workflow.ReconcileHistory(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to reconcile history: %w", ErrTransient, err)
		}
	}

	if plan.QueryChanged {
		d.workflow.SetQueryChanged(true)
	}

	// 6. Output Decision
	// We force shouldWrite to true (meaning "should persist") if:
	// - diff.HasChanges(): Actual data differences (Added/Removed/Modified) exist.
	// - plan.IsColdStart: This is the first run. We must persist the initial StateBytes
	//   as a baseline, even if the generated diff is empty (no changes relative to "nothing").
	// - plan.QueryChanged: The query signature changed. We must persist the new StateBytes
	//   linked to the new query so future runs compare against the correct context,
	//   even if the feature list data happens to be identical.
	shouldWrite := d.workflow.HasChanges() || plan.IsColdStart || plan.QueryChanged
	if !shouldWrite {
		return nil, ErrNoChangesDetected
	}

	finalDiff := d.workflow.GetDiff()
	newStateID := d.idGenerator.NewStateID()
	diffID := d.idGenerator.NewDiffID()

	diffBytes, err := d.diffSerializer.Serialize(diffID, searchID, eventID, newStateID,
		prevCtx.ID, finalDiff, d.timeNow())
	if err != nil {
		return nil, fmt.Errorf("%w, failed to serialize diff: %w", ErrFatal, err)
	}

	newStateBytes, err := d.stateAdapter.Serialize(newStateID, searchID, eventID, query, d.timeNow(), data.NewSnapshot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize new state: %w", ErrFatal, err)
	}

	summaryBytes, err := d.workflow.GenerateJSONSummary()
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	return &DiffResult{
		State:   BlobArtifact{ID: newStateID, Bytes: newStateBytes},
		Diff:    BlobArtifact{ID: eventID, Bytes: diffBytes},
		Summary: summaryBytes,
		Reasons: d.determineReasons(plan),
	}, nil
}

func (d *FeatureDiffer[D]) determineReasons(plan executionPlan) []workertypes.Reason {
	var reasons []workertypes.Reason
	if plan.QueryChanged {
		reasons = append(reasons, workertypes.ReasonQueryChanged)
	}

	if d.workflow.HasDataChanges() {
		reasons = append(reasons, workertypes.ReasonDataUpdated)
	}

	return reasons
}

// --- Internal Helper: Context Loading ---

type previousContext struct {
	Signature string
	Snapshot  map[string]comparables.Feature
	IsEmpty   bool
	ID        string
}

// --- Internal Helper: Planning ---

type executionPlan struct {
	IsColdStart   bool
	QueryChanged  bool
	CurrentQuery  string
	PreviousQuery string
}

func (d *FeatureDiffer[D]) determinePlan(currentQuery string, prev previousContext) executionPlan {
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
	OldSnapshot    map[string]comparables.Feature
	TargetSnapshot map[string]comparables.Feature
	NewSnapshot    map[string]comparables.Feature
}

func (d *FeatureDiffer[D]) executePlan(ctx context.Context, plan executionPlan) (executionData, error) {
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

func toSnapshot(features []backend.Feature) map[string]comparables.Feature {
	m := make(map[string]comparables.Feature)
	for _, f := range features {
		m[f.FeatureId] = toComparable(f)
	}

	return m
}

func toComparable(f backend.Feature) comparables.Feature {
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

	baseline := comparables.BaselineState{
		Status:   generic.OptionallySet[backend.BaselineInfoStatus]{Value: status, IsSet: true},
		LowDate:  generic.OptionallySet[*time.Time]{Value: lowDate, IsSet: true},
		HighDate: generic.OptionallySet[*time.Time]{Value: highDate, IsSet: true},
	}

	cf := comparables.Feature{
		ID:             f.FeatureId,
		Name:           generic.OptionallySet[string]{Value: f.Name, IsSet: true},
		BaselineStatus: generic.OptionallySet[comparables.BaselineState]{Value: baseline, IsSet: true},
		// TODO: Handle Docs when https://github.com/GoogleChrome/webstatus.dev/issues/930 is supported.
		Docs:         generic.UnsetOpt[comparables.Docs](),
		BrowserImpls: generic.UnsetOpt[comparables.BrowserImplementations](),
	}

	if f.BrowserImplementations == nil {
		return cf
	}

	raw := *f.BrowserImplementations
	cf.BrowserImpls = generic.OptionallySet[comparables.BrowserImplementations]{
		Value: comparables.BrowserImplementations{
			Chrome:         toComparableBrowserState(raw[string(backend.Chrome)]),
			ChromeAndroid:  toComparableBrowserState(raw[string(backend.ChromeAndroid)]),
			Edge:           toComparableBrowserState(raw[string(backend.Edge)]),
			Firefox:        toComparableBrowserState(raw[string(backend.Firefox)]),
			FirefoxAndroid: toComparableBrowserState(raw[string(backend.FirefoxAndroid)]),
			Safari:         toComparableBrowserState(raw[string(backend.Safari)]),
			SafariIos:      toComparableBrowserState(raw[string(backend.SafariIos)]),
		},
		IsSet: true,
	}

	return cf
}

// toComparableBrowserState converts a single browser implementation from the backend API
// into the canonical comparable format.
func toComparableBrowserState(impl backend.BrowserImplementation) generic.OptionallySet[comparables.BrowserState] {
	var status backend.BrowserImplementationStatus
	if impl.Status != nil {
		status = *impl.Status
	}

	var date *time.Time
	if impl.Date != nil {
		date = &impl.Date.Time
	}

	// An empty struct from the map lookup indicates the browser was not present.
	// In this case, we return an unset OptionallySet.
	if impl.Status == nil && impl.Date == nil && impl.Version == nil {
		return generic.UnsetOpt[comparables.BrowserState]()
	}

	return generic.OptionallySet[comparables.BrowserState]{
		Value: comparables.BrowserState{
			Status:  generic.OptionallySet[backend.BrowserImplementationStatus]{Value: status, IsSet: true},
			Version: generic.OptionallySet[*string]{Value: impl.Version, IsSet: impl.Version != nil},
			Date:    generic.OptionallySet[*time.Time]{Value: date, IsSet: date != nil},
		},
		IsSet: true,
	}
}

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
	workflow := d.workflowFactory()
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
		workflow.CalculateDiff(data.OldSnapshot, data.TargetSnapshot)
	}

	// 5. Reconcile History
	if workflow.HasRemovedFeatures() && !plan.IsColdStart {
		err = workflow.ReconcileHistory(ctx, data.OldSnapshot, data.NewSnapshot)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to reconcile history: %w", ErrTransient, err)
		}
	}

	if plan.QueryChanged {
		workflow.SetQueryChanged(true)
	}

	// 6. Output Decision
	// We force shouldWrite to true (meaning "should persist") if:
	// - diff.HasChanges(): Actual data differences (Added/Removed/Modified) exist.
	// - plan.IsColdStart: This is the first run. We must persist the initial StateBytes
	//   as a baseline, even if the generated diff is empty (no changes relative to "nothing").
	// - plan.QueryChanged: The query signature changed. We must persist the new StateBytes
	//   linked to the new query so future runs compare against the correct context,
	//   even if the feature list data happens to be identical.
	shouldWrite := workflow.HasChanges() || plan.IsColdStart || plan.QueryChanged
	if !shouldWrite {
		return nil, ErrNoChangesDetected
	}

	finalDiff := workflow.GetDiff()
	newStateID := d.idGenerator.NewStateID()
	diffID := d.idGenerator.NewDiffID()

	t := d.timeNow()

	diffBytes, err := d.diffSerializer.Serialize(diffID, searchID, eventID, newStateID,
		prevCtx.ID, finalDiff, t)
	if err != nil {
		return nil, fmt.Errorf("%w, failed to serialize diff: %w", ErrFatal, err)
	}

	newStateBytes, err := d.stateAdapter.Serialize(newStateID, searchID, eventID, query, t, data.NewSnapshot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize new state: %w", ErrFatal, err)
	}

	summaryBytes, err := workflow.GenerateJSONSummary()
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	return &DiffResult{
		State:       BlobArtifact{ID: newStateID, Bytes: newStateBytes},
		Diff:        BlobArtifact{ID: eventID, Bytes: diffBytes},
		Summary:     summaryBytes,
		Reasons:     d.determineReasons(plan, workflow),
		GeneratedAt: t,
	}, nil
}

func (d *FeatureDiffer[D]) determineReasons(plan executionPlan, workflow StateCompareWorkflow[D]) []workertypes.Reason {
	var reasons []workertypes.Reason
	if plan.QueryChanged {
		reasons = append(reasons, workertypes.ReasonQueryChanged)
	}

	if workflow.HasDataChanges() {
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
	data.NewSnapshot = comparables.NewFeatureMapFromBackendFeatures(newLive)

	if plan.IsColdStart {
		return data, nil
	}

	if plan.QueryChanged {
		oldLive, err := d.client.FetchFeatures(ctx, plan.PreviousQuery)
		if err == nil {
			data.TargetSnapshot = comparables.NewFeatureMapFromBackendFeatures(oldLive)
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

package differ

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// FeatureFetcher abstracts the external API.
type FeatureFetcher interface {
	FetchFeatures(ctx context.Context, query string) ([]backend.Feature, error)
	GetFeature(ctx context.Context, featureID string) (*backendtypes.GetFeatureResult, error)
}

type FeatureDiffer struct {
	client   FeatureFetcher
	migrator *blobtypes.Migrator
}

func NewFeatureDiffer(client FeatureFetcher) *FeatureDiffer {
	m := blobtypes.NewMigrator()

	return &FeatureDiffer{
		client:   client,
		migrator: m,
	}
}

// Run executes the core diffing pipeline.
func (d *FeatureDiffer) Run(ctx context.Context, searchID string, query string,
	previousStateBytes []byte) ([]byte, *FeatureDiff, bool, error) {
	// 1. Load Context
	prevCtx, err := d.loadPreviousContext(previousStateBytes)
	if err != nil {
		return nil, nil, false, fmt.Errorf("%w: failed to load previous state: %w", ErrFatal, err)
	}

	// 2. Plan
	plan := d.determinePlan(query, prevCtx)

	// 3. Execute Fetch
	data, err := d.executePlan(ctx, plan)
	if err != nil {
		return nil, nil, false, fmt.Errorf("%w: failed to fetch data: %w", ErrTransient, err)
	}
	data.OldSnapshot = prevCtx.Snapshot

	// 4. Compute Pure Diff
	var diff *FeatureDiff
	// We check data.TargetSnapshot != nil because if the Flush Strategy failed (in executePlan),
	// it returns nil to signal "Skip Diffing".
	// toSnapshot() guarantees a non-nil map (empty map) for valid empty results,
	// so nil strictly means "Data Not Available".
	if !plan.IsColdStart && data.TargetSnapshot != nil {
		diff = calculateDiff(data.OldSnapshot, data.TargetSnapshot)
	} else {
		diff = new(FeatureDiff)
	}

	// 5. Reconcile History
	if len(diff.Removed) > 0 && !plan.IsColdStart {
		diff, err = d.reconcileHistory(ctx, diff)
		if err != nil {
			return nil, nil, false, fmt.Errorf("%w: failed to reconcile history: %w", ErrTransient, err)
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
	shouldWrite := diff.HasChanges() || plan.IsColdStart || plan.QueryChanged
	if !shouldWrite {
		return nil, nil, false, nil
	}

	newStateBytes, err := d.serializeState(searchID, query, data.NewSnapshot)
	if err != nil {
		return nil, nil, false, fmt.Errorf("%w: failed to serialize new state: %w", ErrFatal, err)
	}

	return newStateBytes, diff, true, nil
}

func (d *FeatureDiffer) serializeState(searchID, query string, snapshot map[string]ComparableFeature) ([]byte, error) {
	payload := FeatureListSnapshot{
		Metadata: StateMetadata{
			GeneratedAt:    time.Now(),
			SearchID:       searchID,
			QuerySignature: query,
		},
		Data: FeatureListData{
			Features: snapshot,
		},
	}

	return blobtypes.NewBlob(payload)
}

// --- Internal Helper: Context Loading ---

type previousContext struct {
	Signature string
	Snapshot  map[string]ComparableFeature
	IsEmpty   bool
}

func (d *FeatureDiffer) loadPreviousContext(bytes []byte) (previousContext, error) {
	if len(bytes) == 0 {
		return previousContext{IsEmpty: true, Signature: "", Snapshot: nil}, nil
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
		Signature: snapshot.Metadata.QuerySignature,
		Snapshot:  snapshot.Data.Features,
		IsEmpty:   false,
	}, nil
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
	if f.Baseline != nil && f.Baseline.Status != nil {
		status = *f.Baseline.Status
	}
	cf := ComparableFeature{
		ID:             f.FeatureId,
		Name:           OptionallySet[string]{Value: f.Name, IsSet: true},
		BaselineStatus: OptionallySet[backend.BaselineInfoStatus]{Value: status, IsSet: true},
		BrowserImpls: BrowserImplementations{
			Chrome:         OptionallySet[string]{Value: "", IsSet: false},
			ChromeAndroid:  OptionallySet[string]{Value: "", IsSet: false},
			Edge:           OptionallySet[string]{Value: "", IsSet: false},
			Firefox:        OptionallySet[string]{Value: "", IsSet: false},
			FirefoxAndroid: OptionallySet[string]{Value: "", IsSet: false},
			Safari:         OptionallySet[string]{Value: "", IsSet: false},
			SafariIos:      OptionallySet[string]{Value: "", IsSet: false},
		},
	}

	// Manually map known browsers from the map to the struct.
	// This hardcoding is intentional: it ensures we only track what we have defined in the struct,
	// allowing us to control schema evolution via struct updates.
	if f.BrowserImplementations != nil {
		raw := *f.BrowserImplementations
		getStatus := func(key string) OptionallySet[string] {
			if impl, ok := raw[key]; ok && impl.Status != nil {
				return OptionallySet[string]{Value: string(*impl.Status), IsSet: true}
			}

			return OptionallySet[string]{Value: "", IsSet: false}
		}

		// Map to struct fields using backend constants or strings
		cf.BrowserImpls.Chrome = getStatus(string(backend.Chrome))
		cf.BrowserImpls.ChromeAndroid = getStatus(string(backend.ChromeAndroid))
		cf.BrowserImpls.Edge = getStatus(string(backend.Edge))
		cf.BrowserImpls.Firefox = getStatus(string(backend.Firefox))
		cf.BrowserImpls.FirefoxAndroid = getStatus(string(backend.FirefoxAndroid))
		cf.BrowserImpls.Safari = getStatus(string(backend.Safari))
		cf.BrowserImpls.SafariIos = getStatus(string(backend.SafariIos))
	}

	return cf
}

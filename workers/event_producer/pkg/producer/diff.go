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
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/blobtypes"
	featurelistv1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelist/v1"
	featurelistdiffv1 "github.com/GoogleChrome/webstatus.dev/lib/blobtypes/featurelistdiff/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/comparables"
	"github.com/GoogleChrome/webstatus.dev/workers/event_producer/pkg/differ"
)

var ErrInvalidFormat = errors.New("invalid format")

// FeatureFetcher abstracts the external API.
type FeatureFetcher interface {
	FetchFeatures(ctx context.Context, query string) ([]backend.Feature, error)
	GetFeature(ctx context.Context, featureID string) (*backendtypes.GetFeatureResult, error)
}

type migratorFunc func(bytes []byte) ([]byte, error)

// stateConverter is a generic function type that defines how to convert a
// versioned state snapshot of type S into the canonical diffing format.
type stateConverter[S any] func(state *S) (map[string]comparables.Feature, string)

// stateSerializerFunc defines how to create a versioned state snapshot `S`
// from the canonical feature map and serialize it into raw bytes.
type stateSerializerFunc[S any] func(id, searchID, eventID, query string,
	snapshot map[string]comparables.Feature, timestamp time.Time) ([]byte, error)

// v1StateSerializerFunc implements StateSerializerFunc for v1.FeatureListSnapshot.
func v1StateSerializerFunc(id, searchID, eventID, query string,
	snapshot map[string]comparables.Feature, timestamp time.Time) ([]byte, error) {
	// Convert the canonical comparables.Feature map back to v1.Feature map
	// This is the inverse of convertV1SnapshotToComparable logic.
	v1Features := make(map[string]featurelistv1.Feature, len(snapshot))
	for id, comparableFeature := range snapshot {
		v1Features[id] = convertComparableToV1Feature(comparableFeature)
	}

	payload := featurelistv1.FeatureListSnapshot{
		Metadata: featurelistv1.StateMetadata{
			GeneratedAt:    timestamp,
			SearchID:       searchID,
			QuerySignature: query,
			ID:             id,
			EventID:        eventID,
		},
		Data: featurelistv1.FeatureListData{
			Features: v1Features,
		},
	}

	return blobtypes.NewBlob(payload)
}

// genericStateAdapter is a single, reusable implementation of the differ.StateAdapter interface.
// It is generic over the state snapshot type S.
type genericStateAdapter[S snapshot] struct {
	migrator   migratorFunc
	converter  stateConverter[S]
	serializer stateSerializerFunc[S]
}

// newGenericStateAdapter creates the adapter, injecting the specific converter it should use.
func newGenericStateAdapter[S snapshot](
	migrator migratorFunc,
	converter stateConverter[S],
	serializer stateSerializerFunc[S],
) *genericStateAdapter[S] {
	return &genericStateAdapter[S]{
		migrator:   migrator,
		converter:  converter,
		serializer: serializer,
	}
}

type snapshot interface {
	ID() string
}

// Load implements the differ.StateAdapter interface.
func (a *genericStateAdapter[S]) Load(bytes []byte) (
	map[string]comparables.Feature, string, string, bool, error,
) {
	if len(bytes) == 0 {
		return nil, "", "", true, nil
	}
	migratedBytes, err := a.migrator(bytes)
	if err != nil {
		return nil, "", "", false, err
	}

	// 1. Unmarshal into the generic type S.
	// We declare a variable of type S, and Go's generics ensure it's the correct concrete struct.
	var snapshot S
	if err := json.Unmarshal(migratedBytes, &snapshot); err != nil {
		return nil, "", "", false, errors.Join(err, ErrInvalidFormat)
	}

	// 2. Use the injected converter function to perform the translation.
	compMap, signature := a.converter(&snapshot)

	// 3. Return the canonical data.
	return compMap, snapshot.ID(), signature, false, nil
}

func (a *genericStateAdapter[S]) Serialize(id, searchID, eventID, query string,
	timestamp time.Time, snapshot map[string]comparables.Feature) ([]byte, error) {
	return a.serializer(id, searchID, eventID, query, snapshot, timestamp)
}

// V1DiffSerializer is a concrete implementation for serializing V1 diffs.
type V1DiffSerializer struct{}

// NewV1DiffSerializer creates a new serializer for V1 diffs.
func NewV1DiffSerializer() *V1DiffSerializer {
	return &V1DiffSerializer{}
}

// Serialize implements the differ.DiffSerializer interface.
// It takes the pure V1 diff data and wraps it in the versioned blob envelope.
func (s *V1DiffSerializer) Serialize(
	id, searchID, eventID, newStateID, previousStateID string, diff *featurelistdiffv1.FeatureDiff, timestamp time.Time,
) ([]byte, error) {

	payload := featurelistdiffv1.FeatureDiffSnapshot{
		Metadata: featurelistdiffv1.DiffMetadata{
			ID:              id,
			EventID:         eventID,
			SearchID:        searchID,
			NewStateID:      newStateID,
			PreviousStateID: previousStateID,
			GeneratedAt:     timestamp,
		},
		Data: *diff,
	}

	return blobtypes.NewBlob(payload)
}

// convertV1SnapshotToComparable matches the StateConverter[featurelistv1.FeatureListSnapshot] signature.
func convertV1SnapshotToComparable(state *featurelistv1.FeatureListSnapshot) (map[string]comparables.Feature, string) {
	comparableMap := make(map[string]comparables.Feature, len(state.Data.Features))
	for id, v1Feature := range state.Data.Features {
		comparableMap[id] = convertV1FeatureToComparable(v1Feature)
	}

	return comparableMap, state.Metadata.QuerySignature
}

func NewDiffer(client FeatureFetcher) *differ.FeatureDiffer[featurelistdiffv1.FeatureDiff] {
	m := blobtypes.NewMigrator()
	// In the future, do the registration for migration here
	v1MigrationFunc := func(bytes []byte) ([]byte, error) {
		return blobtypes.Apply[featurelistv1.FeatureListSnapshot](m, bytes)
	}

	stateAdapter := newGenericStateAdapter(v1MigrationFunc, convertV1SnapshotToComparable, v1StateSerializerFunc)

	diffSerializer := NewV1DiffSerializer()
	workflow := featurelistdiffv1.NewFeatureDiffWorkflow(client, &workertypes.FeatureDiffV1SummaryGenerator{})

	return differ.NewFeatureDiffer[featurelistdiffv1.FeatureDiff](client, workflow, stateAdapter, diffSerializer)
}

// convertV1FeatureToComparable maps a V1 feature struct to the canonical comparables.Feature.
func convertV1FeatureToComparable(v1f featurelistv1.Feature) comparables.Feature {
	// 1. Convert name and ID
	name := generic.OptionallySet[string]{Value: v1f.Name.Value, IsSet: true}
	id := v1f.ID

	// 2. Convert BaselineStatus
	baselineStatus := generic.UnsetOpt[comparables.BaselineState]()
	if v1f.BaselineStatus.IsSet {
		baselineStatus.IsSet = true
		baselineInfoStatus := generic.UnsetOpt[backend.BaselineInfoStatus]()
		if v1f.BaselineStatus.Value.Status.IsSet {
			baselineInfoStatus.IsSet = true
			var status backend.BaselineInfoStatus
			switch v1f.BaselineStatus.Value.Status.Value {
			case featurelistv1.Limited:
				status = backend.Limited
			case featurelistv1.Newly:
				status = backend.Newly
			case featurelistv1.Widely:
				status = backend.Widely
			}
			baselineInfoStatus.Value = status
			baselineStatus.Value.Status = baselineInfoStatus
		}
		baselineStatus.Value.LowDate = v1f.BaselineStatus.Value.LowDate
		baselineStatus.Value.HighDate = v1f.BaselineStatus.Value.HighDate

	}

	// 3. Convert BrowserImplementations
	browserImpls := generic.UnsetOpt[comparables.BrowserImplementations]()
	if v1f.BrowserImpls.IsSet {
		browserImpls.IsSet = true
		browserImpls.Value = comparables.BrowserImplementations{
			Chrome:         convertV1BrowserStatusToComparableState(v1f.BrowserImpls.Value.Chrome),
			ChromeAndroid:  convertV1BrowserStatusToComparableState(v1f.BrowserImpls.Value.ChromeAndroid),
			Edge:           convertV1BrowserStatusToComparableState(v1f.BrowserImpls.Value.Edge),
			Firefox:        convertV1BrowserStatusToComparableState(v1f.BrowserImpls.Value.Firefox),
			FirefoxAndroid: convertV1BrowserStatusToComparableState(v1f.BrowserImpls.Value.FirefoxAndroid),
			Safari:         convertV1BrowserStatusToComparableState(v1f.BrowserImpls.Value.Safari),
			SafariIos:      convertV1BrowserStatusToComparableState(v1f.BrowserImpls.Value.SafariIos),
		}
	}

	// 4. Convert Docs
	docs := generic.UnsetOpt[comparables.Docs]()
	if v1f.Docs.IsSet {
		docs.IsSet = true
		mdnDocs := generic.UnsetOpt[[]comparables.MdnDoc]()
		if v1f.Docs.Value.MdnDocs.IsSet {
			mdnDocs.IsSet = true
			for _, v1Doc := range v1f.Docs.Value.MdnDocs.Value {
				mdnDocs.Value = append(mdnDocs.Value, comparables.MdnDoc{
					URL:   v1Doc.URL,
					Title: v1Doc.Title,
					Slug:  v1Doc.Slug,
				})
			}
		}
		docs.Value.MdnDocs = mdnDocs
	}

	return comparables.Feature{
		ID:             id,
		Name:           name,
		BaselineStatus: baselineStatus,
		BrowserImpls:   browserImpls,
		Docs:           docs,
	}
}

// convertV1BrowserStatusToComparableState converts a simple string status from V1
// into the more detailed comparables.BrowserState.
func convertV1BrowserStatusToComparableState(
	state generic.OptionallySet[featurelistv1.BrowserState]) generic.OptionallySet[comparables.BrowserState] {
	browserState := generic.UnsetOpt[comparables.BrowserState]()
	if !state.IsSet {
		return browserState
	}
	browserState.IsSet = true
	browserImplStatus := generic.UnsetOpt[backend.BrowserImplementationStatus]()
	var status backend.BrowserImplementationStatus
	if state.Value.Status.IsSet {
		switch state.Value.Status.Value {
		case featurelistv1.Available:
			status = backend.Available
		case featurelistv1.Unavailable:
			status = backend.Unavailable
		}
		browserImplStatus.IsSet = true
		browserImplStatus.Value = status
		browserState.Value.Status = browserImplStatus
	}
	browserState.Value.Version = state.Value.Version
	browserState.Value.Date = state.Value.Date

	return browserState
}

// convertComparableToV1Feature maps a canonical comparables.Feature to a V1 feature struct.
func convertComparableToV1Feature(cf comparables.Feature) featurelistv1.Feature {
	// 1. Convert name and ID
	name := generic.OptionallySet[string]{Value: cf.Name.Value, IsSet: cf.Name.IsSet}
	id := cf.ID

	// 2. Convert BaselineStatus
	baselineStatus := generic.UnsetOpt[featurelistv1.BaselineState]()
	if cf.BaselineStatus.IsSet {
		baselineStatus.IsSet = true
		baselineInfoStatus := generic.UnsetOpt[featurelistv1.BaselineInfoStatus]()
		if cf.BaselineStatus.Value.Status.IsSet {
			baselineInfoStatus.IsSet = true
			var status featurelistv1.BaselineInfoStatus
			switch cf.BaselineStatus.Value.Status.Value {
			case backend.Limited:
				status = featurelistv1.Limited
			case backend.Newly:
				status = featurelistv1.Newly
			case backend.Widely:
				status = featurelistv1.Widely
			}
			baselineInfoStatus.Value = status
			baselineStatus.Value.Status = baselineInfoStatus
		}
		baselineStatus.Value.LowDate = cf.BaselineStatus.Value.LowDate
		baselineStatus.Value.HighDate = cf.BaselineStatus.Value.HighDate
	}

	// 3. Convert BrowserImplementations
	browserImpls := generic.UnsetOpt[featurelistv1.BrowserImplementations]()
	if cf.BrowserImpls.IsSet {
		browserImpls.IsSet = true
		browserImpls.Value = featurelistv1.BrowserImplementations{
			Chrome:         convertComparableBrowserStateToV1(cf.BrowserImpls.Value.Chrome),
			ChromeAndroid:  convertComparableBrowserStateToV1(cf.BrowserImpls.Value.ChromeAndroid),
			Edge:           convertComparableBrowserStateToV1(cf.BrowserImpls.Value.Edge),
			Firefox:        convertComparableBrowserStateToV1(cf.BrowserImpls.Value.Firefox),
			FirefoxAndroid: convertComparableBrowserStateToV1(cf.BrowserImpls.Value.FirefoxAndroid),
			Safari:         convertComparableBrowserStateToV1(cf.BrowserImpls.Value.Safari),
			SafariIos:      convertComparableBrowserStateToV1(cf.BrowserImpls.Value.SafariIos),
		}
	}

	// 4. Convert Docs
	docs := generic.UnsetOpt[featurelistv1.Docs]()
	if cf.Docs.IsSet {
		docs.IsSet = true
		mdnDocs := generic.UnsetOpt[[]featurelistv1.MdnDoc]()
		if cf.Docs.Value.MdnDocs.IsSet {
			mdnDocs.IsSet = true
			for _, compDoc := range cf.Docs.Value.MdnDocs.Value {
				mdnDocs.Value = append(mdnDocs.Value, featurelistv1.MdnDoc{
					URL:   compDoc.URL,
					Title: compDoc.Title,
					Slug:  compDoc.Slug,
				})
			}
		}
		docs.Value.MdnDocs = mdnDocs
	}

	return featurelistv1.Feature{
		ID:             id,
		Name:           name,
		BaselineStatus: baselineStatus,
		BrowserImpls:   browserImpls,
		Docs:           docs,
	}
}

// convertComparableBrowserStateToV1 converts a comparables.BrowserState to a V1 featurelistv1.BrowserState.
func convertComparableBrowserStateToV1(
	state generic.OptionallySet[comparables.BrowserState]) generic.OptionallySet[featurelistv1.BrowserState] {
	v1BrowserState := generic.UnsetOpt[featurelistv1.BrowserState]()
	if !state.IsSet {
		return v1BrowserState
	}
	v1BrowserState.IsSet = true
	v1BrowserImplStatus := generic.UnsetOpt[featurelistv1.BrowserImplementationStatus]()
	var status featurelistv1.BrowserImplementationStatus
	if state.Value.Status.IsSet {
		switch state.Value.Status.Value {
		case backend.Available:
			status = featurelistv1.Available
		case backend.Unavailable:
			status = featurelistv1.Unavailable
		}
		v1BrowserImplStatus.IsSet = true
		v1BrowserImplStatus.Value = status
		v1BrowserState.Value.Status = v1BrowserImplStatus
	}
	v1BrowserState.Value.Version = state.Value.Version
	v1BrowserState.Value.Date = state.Value.Date

	return v1BrowserState
}

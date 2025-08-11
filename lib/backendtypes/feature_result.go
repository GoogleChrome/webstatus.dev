package backendtypes

import "github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"

type resultType string

const (
	regularFeatureType resultType = "regular"
	splitFeatureType   resultType = "split"
	movedFeatureType   resultType = "moved"
)

// GetFeatureResult contains the different types of feature results when getting
// a specific feature by feature ID.
type GetFeatureResult struct {
	feature        *backend.Feature
	movedFeatureID *string
	splitFeature   *backend.FeatureSplitInfo
	resultType     resultType
}

// NewRegularFeatureResult creates a new GetFeatureResult for a regular feature.
func NewRegularFeatureResult(feature *backend.Feature) *GetFeatureResult {
	return &GetFeatureResult{
		feature:        feature,
		movedFeatureID: nil,
		splitFeature:   nil,
		resultType:     regularFeatureType,
	}
}

// NewMovedFeatureResult creates a new GetFeatureResult for a moved feature.
func NewMovedFeatureResult(movedFeatureID string) *GetFeatureResult {
	return &GetFeatureResult{
		feature:        nil,
		movedFeatureID: &movedFeatureID,
		splitFeature:   nil,
		resultType:     movedFeatureType,
	}
}

// NewSplitFeatureResult creates a new GetFeatureResult for a split feature.
func NewSplitFeatureResult(splitFeature *backend.FeatureSplitInfo) *GetFeatureResult {
	return &GetFeatureResult{
		feature:        nil,
		movedFeatureID: nil,
		splitFeature:   splitFeature,
		resultType:     splitFeatureType,
	}
}

func (r GetFeatureResult) IsRegular() bool {
	return r.resultType == regularFeatureType && r.feature != nil
}

func (r GetFeatureResult) IsMoved() bool {
	return r.resultType == movedFeatureType && r.movedFeatureID != nil
}

func (r GetFeatureResult) IsSplit() bool {
	return r.resultType == splitFeatureType && r.splitFeature != nil
}

func (r GetFeatureResult) GetFeature() *backend.Feature {
	return r.feature
}

func (r GetFeatureResult) GetMovedFeatureID() *string {
	return r.movedFeatureID
}

func (r GetFeatureResult) GetSplitFeature() *backend.FeatureSplitInfo {
	return r.splitFeature
}

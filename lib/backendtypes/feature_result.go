package backendtypes

import (
	"context"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// FeatureResultVisitor defines the methods for visiting each result type.
type FeatureResultVisitor interface {
	VisitRegularFeature(ctx context.Context, result RegularFeatureResult) error
	VisitMovedFeature(ctx context.Context, result MovedFeatureResult) error
	VisitSplitFeature(ctx context.Context, result SplitFeatureResult) error
}

// RegularFeatureResult represents a result for a regular feature.
type RegularFeatureResult struct {
	feature *backend.Feature
}

func (r RegularFeatureResult) Visit(ctx context.Context, v FeatureResultVisitor) error {
	return v.VisitRegularFeature(ctx, r)
}

func (r RegularFeatureResult) Feature() *backend.Feature {
	return r.feature
}

// RegularFeatureResult creates a new RegularFeatureResult for a regular feature.
func NewRegularFeatureResult(feature *backend.Feature) *RegularFeatureResult {
	return &RegularFeatureResult{
		feature: feature,
	}
}

// SplitFeatureResult represents a result for a split feature.
type SplitFeatureResult struct {
	splitFeature backend.FeatureEvolutionSplit
}

func (s SplitFeatureResult) Visit(ctx context.Context, v FeatureResultVisitor) error {
	return v.VisitSplitFeature(ctx, s)
}

func (s SplitFeatureResult) SplitFeature() backend.FeatureEvolutionSplit {
	return s.splitFeature
}

// NewSplitFeatureResult creates a new SplitFeatureResult for a split feature.
func NewSplitFeatureResult(splitFeature backend.FeatureEvolutionSplit) *SplitFeatureResult {
	return &SplitFeatureResult{
		splitFeature: splitFeature,
	}
}

// MovedFeatureResult represents a result for a moved feature.
type MovedFeatureResult struct {
	newFeatureID string
}

func (m MovedFeatureResult) Visit(ctx context.Context, v FeatureResultVisitor) error {
	return v.VisitMovedFeature(ctx, m)
}

// NewMovedFeatureResult creates a new MovedFeatureResult for a moved feature.
func NewMovedFeatureResult(newFeatureID string) *MovedFeatureResult {
	return &MovedFeatureResult{
		newFeatureID: newFeatureID,
	}
}

func (m MovedFeatureResult) NewFeatureID() string {
	return m.newFeatureID
}

// FeatureResult is the interface that all concrete results implement.
// The Visit method allows a visitor to operate on the concrete type.
type FeatureResult interface {
	Visit(ctx context.Context, visitor FeatureResultVisitor) error
}

// GetFeatureResult is a container for the result of a GetFeature operation.
type GetFeatureResult struct {
	result FeatureResult
}

// Visit allows a visitor to operate on the result.
func (g GetFeatureResult) Visit(ctx context.Context, v FeatureResultVisitor) error {
	return g.result.Visit(ctx, v)
}

// NewGetFeatureResult creates a new GetFeatureResult.
func NewGetFeatureResult(result FeatureResult) *GetFeatureResult {
	return &GetFeatureResult{
		result: result,
	}
}

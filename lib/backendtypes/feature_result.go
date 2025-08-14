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

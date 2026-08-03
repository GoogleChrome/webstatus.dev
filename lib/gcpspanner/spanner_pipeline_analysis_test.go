// Copyright 2024 Google LLC
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

package gcpspanner

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
)

// TestWorstCaseMaxComplexityQuerySpannerLimits verifies that queries at the maximum allowed
// AST node complexity (50 nodes) remain well within Cloud Spanner parameter (950) and
// SQL statement size (1 MB) limits.
func TestWorstCaseMaxComplexityQuerySpannerLimits(t *testing.T) {
	terms := []string{
		"id:feat-01", "id:feat-02", "id:feat-03", "id:feat-04", "id:feat-05",
		"group:css", "group:layout", "group:dom", "group:canvas", "group:webgl",
		"snapshot:2023", "snapshot:2024", "snapshot:ecmascript-5",
		"available_on:chrome", "available_on:firefox", "available_on:safari",
		"baseline_status:widely", "baseline_status:newly",
	}

	for len(terms) < 24 {
		terms = append(terms, fmt.Sprintf("id:extra-feat-%d", len(terms)))
	}

	rawQuery := fmt.Sprintf("(%s)", strings.Join(terms, " OR "))

	parser := searchtypes.FeaturesSearchQueryParser{}
	astNode, err := parser.Parse(rawQuery)
	if err != nil {
		t.Fatalf("Failed to parse worst-case query: %v", err)
	}

	dedupNode := searchtypes.Deduplicate(astNode)
	nodeCount := searchtypes.CountNodes(dedupNode)
	if nodeCount > 50 {
		t.Fatalf("expected <= 50 AST nodes, got %d", nodeCount)
	}

	builder := NewFeatureSearchFilterBuilder()
	compiledFilter, err := builder.Build(dedupNode)
	if err != nil {
		t.Fatalf("Failed to build compiled filter: %v", err)
	}

	queryBuilder := FeatureSearchQueryBuilder{
		baseQuery:     GCPFeatureSearchBaseQuery{},
		wptMetricView: WPTSubtestView,
		browsers:      []string{"chrome", "firefox", "safari"},
		offsetCursor:  nil,
	}

	sort := NewFeatureNameSort(true)
	stmt := queryBuilder.Build(compiledFilter, sort, 25)

	// Parameter count assertion: Spanner hard limit is 950
	if len(stmt.Params) > 100 {
		t.Errorf("Param count %d exceeds safety threshold of 100 (Spanner max 950)", len(stmt.Params))
	}

	// SQL statement size assertion: Spanner hard limit is 1MB (1,048,576 bytes)
	if len(stmt.SQL) > 50000 {
		t.Errorf("SQL length %d bytes exceeds safety threshold of 50KB (Spanner max 1MB)", len(stmt.SQL))
	}
}

func TestSpannerQueryGeneration35Features(t *testing.T) {
	var ids []string
	for i := 1; i <= 35; i++ {
		ids = append(ids, fmt.Sprintf("id:feature-id-%02d", i))
	}
	rawQuery := fmt.Sprintf("(%s) AND baseline_status:widely", strings.Join(ids, " OR "))

	parser := searchtypes.FeaturesSearchQueryParser{}
	astNode, err := parser.Parse(rawQuery)
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	builder := NewFeatureSearchFilterBuilder()
	compiledFilter, err := builder.Build(astNode)
	if err != nil {
		t.Fatalf("Failed to build compiled filter: %v", err)
	}

	queryBuilder := FeatureSearchQueryBuilder{
		baseQuery:     GCPFeatureSearchBaseQuery{},
		wptMetricView: WPTSubtestView,
		browsers:      []string{"chrome", "firefox", "safari"},
		offsetCursor:  nil,
	}

	sort := NewFeatureNameSort(true)
	stmt := queryBuilder.Build(compiledFilter, sort, 25)

	if len(stmt.Params) == 0 {
		t.Error("expected non-empty parameters for 35 feature ID query")
	}
	if len(stmt.SQL) == 0 {
		t.Error("expected non-empty SQL statement for 35 feature ID query")
	}
}

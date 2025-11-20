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

package gh

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-github/v79/github"
)

func TestPaginator(t *testing.T) {
	type TestItem struct {
		Value string
	}

	type LibTestItem struct {
		Name *string
	}

	convertFunc := func(item LibTestItem) (TestItem, bool) {
		if item.Name == nil {
			// nolint:exhaustruct
			return TestItem{}, false
		}

		return TestItem{Value: *item.Name}, true
	}

	var errOnSecondPage = errors.New("api error on second page")

	tests := []struct {
		name          string
		listFunc      func(ctx context.Context, opts *github.ListOptions) ([]LibTestItem, *github.Response, error)
		expectedItems []TestItem
		expectedError error
		expectedPages int
	}{
		{
			name: "Single page success",
			listFunc: func(_ context.Context, _ *github.ListOptions) ([]LibTestItem, *github.Response, error) {
				return []LibTestItem{{
					Name: valuePtr("item1"),
				}}, createTestGithubResponse(0), nil
			},
			expectedItems: []TestItem{{"item1"}},
			expectedError: nil,
			expectedPages: 1,
		},
		{
			name: "Multi-page success",
			listFunc: func(_ context.Context, opts *github.ListOptions) ([]LibTestItem, *github.Response, error) {
				if opts.Page == 1 {
					return []LibTestItem{{
						Name: valuePtr("itemA"),
					}}, createTestGithubResponse(2), nil
				}

				return []LibTestItem{{
					Name: valuePtr("itemB"),
				}}, createTestGithubResponse(0), nil
			},
			expectedItems: []TestItem{{"itemA"}, {"itemB"}},
			expectedError: nil,
			expectedPages: 2,
		},
		{
			name: "API error on first page",
			listFunc: func(_ context.Context, _ *github.ListOptions) ([]LibTestItem, *github.Response, error) {
				return nil, nil, errTestAPI
			},
			expectedItems: nil,
			expectedError: errTestAPI,
			expectedPages: 1,
		},
		{
			name: "API error on second page",
			listFunc: func(_ context.Context, opts *github.ListOptions) ([]LibTestItem, *github.Response, error) {
				if opts.Page == 1 {
					return []LibTestItem{{
						Name: valuePtr("itemA"),
					}}, createTestGithubResponse(2), nil
				}

				return nil, nil, errOnSecondPage
			},
			expectedItems: nil,
			expectedError: errOnSecondPage,
			expectedPages: 2,
		},
		{
			name: "Conversion skips item",
			listFunc: func(_ context.Context, _ *github.ListOptions) ([]LibTestItem, *github.Response, error) {
				return []LibTestItem{
					{Name: valuePtr("valid")},
					{Name: nil},
					{Name: valuePtr("another_valid")},
				}, createTestGithubResponse(0), nil
			},
			expectedItems: []TestItem{{"valid"}, {"another_valid"}},
			expectedError: nil,
			expectedPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPaginator(tt.listFunc, convertFunc)
			var actualItems []TestItem
			var actualError error
			pagesFetched := 0

			for p.HasNextPage() {
				pagesFetched++
				pageItems, err := p.NextPage(context.Background())
				if err != nil {
					actualError = err
					actualItems = nil // Discard partial results on error.

					break
				}
				actualItems = append(actualItems, pageItems...)
			}

			if !errors.Is(actualError, tt.expectedError) {
				t.Errorf("Paginator error = %v, wantErr %v", actualError, tt.expectedError)
			}
			if !reflect.DeepEqual(actualItems, tt.expectedItems) {
				t.Errorf("Paginator items = %v, want %v", actualItems, tt.expectedItems)
			}
			if pagesFetched != tt.expectedPages {
				t.Errorf("Paginator fetched %d pages, want %d", pagesFetched, tt.expectedPages)
			}
		})
	}
}

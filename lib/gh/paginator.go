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

	"github.com/google/go-github/v75/github"
)

// paginator is a generic structure for handling pagination.
type paginator[T any, LibStruct any] struct {
	listFunc    func(ctx context.Context, opts *github.ListOptions) ([]LibStruct, *github.Response, error)
	convert     func(item LibStruct) (T, bool)
	currentPage int
	perPage     int
}

// NextPage fetches the next page of items.
func (p *paginator[T, LibStruct]) NextPage(ctx context.Context) ([]T, error) {
	opts := &github.ListOptions{
		Page:    p.currentPage,
		PerPage: p.perPage,
	}

	items, resp, err := p.listFunc(ctx, opts)
	if err != nil {
		return nil, err
	}

	if resp.NextPage == 0 {
		p.currentPage = 0
	} else {
		p.currentPage = resp.NextPage
	}

	convertedItems := make([]T, 0, len(items))
	for _, item := range items {
		convertedItem, success := p.convert(item)
		if !success {
			continue
		}
		convertedItems = append(convertedItems, convertedItem)
	}

	return convertedItems, nil
}

// newPaginator creates a new Paginator instance.
func newPaginator[T any, LibStruct any](
	listFunc func(ctx context.Context, opts *github.ListOptions) ([]LibStruct, *github.Response, error),
	convert func(item LibStruct) (T, bool)) *paginator[T, LibStruct] {
	return &paginator[T, LibStruct]{
		listFunc:    listFunc,
		currentPage: 1,
		perPage:     100,
		convert:     convert,
	}
}

// HasNextPage checks if there are more pages to fetch.
func (p *paginator[T, LibStruct]) HasNextPage() bool {
	return p.currentPage != 0
}

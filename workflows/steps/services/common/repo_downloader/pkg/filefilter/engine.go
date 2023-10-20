// Copyright 2023 Google LLC
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

package filefilter

import (
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/common/repo_downloader"
)

type Engine struct {
	filters *[]repo_downloader.FileFilter
}

func NewEngine(filters *[]repo_downloader.FileFilter) *Engine {
	return &Engine{filters: filters}
}

func (e *Engine) Applies(filename string) bool {
	// If there are no filters, it applies
	if e.filters == nil || len(*e.filters) == 0 {
		return true
	}

	for _, filter := range *e.filters {
		if filter.Prefix != nil &&
			filter.Suffix != nil &&
			strings.HasPrefix(filename, *filter.Prefix) &&
			strings.HasSuffix(filename, *filter.Suffix) {
			return true
		}
		if filter.Prefix != nil && strings.HasPrefix(filename, *filter.Prefix) && filter.Suffix == nil {
			return true
		}
		if filter.Suffix != nil && strings.HasSuffix(filename, *filter.Suffix) && filter.Prefix == nil {
			return true
		}
	}

	return false
}

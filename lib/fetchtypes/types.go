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

package fetchtypes

import "errors"

var (
	// ErrFailedToBuildRequest indicates the request failed to build. The fetch never occurred.
	ErrFailedToBuildRequest = errors.New("failed to build request")
	// ErrFailedToFetch indicates the fetch failed.
	ErrFailedToFetch = errors.New("failed to fetch")
	// ErrUnexpectedResult indicates the fetch returned an unexpected result.
	ErrUnexpectedResult = errors.New("unexpected result")
	// ErrMissingBody indicates the fetch returned a nil body.
	ErrMissingBody = errors.New("missing body")
)

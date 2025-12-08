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

package blobtypes

import "errors"

// ErrPreconditionFailed indicates the object was modified by another process.
// This happens when the ExpectedGeneration provided to WriteBlob does not match
// the current generation of the object in the store.
// Workers should usually retry the operation (read-modify-write) when this occurs.
var ErrPreconditionFailed = errors.New("blob precondition failed")

// ErrBlobNotFound indicates the requested object does not exist.
// Workers should handle this as a "Cold Start" or empty state.
var ErrBlobNotFound = errors.New("blob not found")

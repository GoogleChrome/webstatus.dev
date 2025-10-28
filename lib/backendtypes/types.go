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

import "errors"

var (
	// ErrInvalidPageToken indicates the page token is invalid.
	// Raised by the adapter layer to let the server layer know that the
	// page token did not meet the expected encoding in the database layer.
	ErrInvalidPageToken = errors.New("invalid page token")

	// ErrUserMaxSavedSearches indicates the user has reached the maximum
	// number of allowed saved searches.
	ErrUserMaxSavedSearches = errors.New("user has reached the maximum number of allowed saved searches")

	// ErrUserMaxBookmarks indicates the user has reached the maximum
	// number of allowed bookmarks.
	ErrUserMaxBookmarks = errors.New("user has reached the maximum number of allowed bookmarks")

	// ErrUserNotAuthorizedForAction indicates the user is not authorized to execute the requested action.
	ErrUserNotAuthorizedForAction = errors.New("user not authorized to execute action")

	// ErrEntityDoesNotExist indicates the entity does not exist.
	ErrEntityDoesNotExist = errors.New("entity does not exist")

	// ErrJSONMarshal indicates a failure when marshalling data from a generic interface{} for conversion.
	// This typically happens during the data conversion from a database type to a JSON byte slice.
	ErrJSONMarshal = errors.New("failed to marshal data for JSON conversion")

	// ErrJSONUnmarshal indicates a failure when unmarshalling JSON data into a target struct.
	// This suggests a mismatch between the data stored in the database and the expected data contract.
	ErrJSONUnmarshal = errors.New("failed to unmarshal JSON data")

	// ErrEmptyJSONValue is a sentinel error indicating that the JSON value from the database
	// was valid but empty (e.g., an empty array or object). This allows callers to distinguish
	// between a missing value and an explicitly empty one.
	ErrEmptyJSONValue = errors.New("JSON value is empty")
)

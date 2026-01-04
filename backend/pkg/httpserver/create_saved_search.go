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

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

const (
	savedSearchNameMaxLength            = 32
	savedSearchNameMinLength            = 1
	savedSearchNameDescriptionMaxLength = 1024
	savedSearchNameDescriptionMinLength = 1
	savedSearchQueryMaxLength           = 256
	savedSearchQueryMinLength           = 1
)

var (
	errSavedSearchInvalidNameLength = fmt.Errorf("name must be between %d and %d characters long",
		savedSearchNameMinLength, savedSearchNameMaxLength)
	errSavedSearchInvalidQueryLength = fmt.Errorf("query must be between %d and %d characters long",
		savedSearchQueryMinLength, savedSearchQueryMaxLength)
	errSavedSearchInvalidDescriptionLength = fmt.Errorf("description must be between %d and %d characters long",
		savedSearchNameDescriptionMinLength, savedSearchNameDescriptionMaxLength)
	errQueryDoesNotMatchGrammar = errors.New("query does not match grammar")
)

type fieldValidationErrors struct {
	fieldErrorMap map[string]string
}

func (f *fieldValidationErrors) addFieldError(field string, err error) {
	if f.fieldErrorMap == nil {
		f.fieldErrorMap = make(map[string]string)
	}
	f.fieldErrorMap[field] = err.Error()
}

func (f fieldValidationErrors) hasErrors() bool {
	return len(f.fieldErrorMap) > 0
}

// validateSavedSearchName checks the validity of the saved search name.
// It expects a pointer to handle potential nil values during updates.
func validateSavedSearchName(name *string, fieldErrors *fieldValidationErrors) {
	if name == nil {
		// Treat nil as invalid if validation is requested
		fieldErrors.addFieldError("name", errSavedSearchInvalidNameLength)

		return
	}
	if len(*name) < savedSearchNameMinLength || len(*name) > savedSearchNameMaxLength {
		fieldErrors.addFieldError("name", errSavedSearchInvalidNameLength)
	}
}

// validateSavedSearchQuery checks the validity of the saved search query.
// It expects a pointer to handle potential nil values during updates.
func validateSavedSearchQuery(query *string, fieldErrors *fieldValidationErrors) {
	if query == nil {
		fieldErrors.addFieldError("query", errSavedSearchInvalidQueryLength)

		return
	}

	if len(*query) < savedSearchQueryMinLength || len(*query) > savedSearchQueryMaxLength {
		fieldErrors.addFieldError("query", errSavedSearchInvalidQueryLength)

		return
	}

	// Only parse if length is okay
	parser := searchtypes.FeaturesSearchQueryParser{}
	_, err := parser.Parse(*query)
	if err != nil {
		fieldErrors.addFieldError("query", errQueryDoesNotMatchGrammar)
	}

}

// validateSavedSearchDescription checks the validity of the saved search description.
// Description is optional, so nil is allowed. Validation only occurs if non-nil.
func validateSavedSearchDescription(description *string, fieldErrors *fieldValidationErrors) {
	// If description is nil, it's considered valid (optional field).
	if description == nil {
		return
	}

	// If description is provided (non-nil), validate its length.
	if len(*description) < savedSearchNameDescriptionMinLength ||
		len(*description) > savedSearchNameDescriptionMaxLength {
		fieldErrors.addFieldError("description", errSavedSearchInvalidDescriptionLength)
	}
}

func validateSavedSearch(input *backend.SavedSearch) *fieldValidationErrors {
	fieldErrors := &fieldValidationErrors{fieldErrorMap: nil}

	// Validate Name (using address of the string field)
	validateSavedSearchName(&input.Name, fieldErrors)

	// Validate Query (using address of the string field)
	validateSavedSearchQuery(&input.Query, fieldErrors)

	// Validate Description (already a pointer)
	validateSavedSearchDescription(input.Description, fieldErrors)

	if fieldErrors.hasErrors() {
		return fieldErrors
	}

	return nil
}

// CreateSavedSearch implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) CreateSavedSearch(ctx context.Context, request backend.CreateSavedSearchRequestObject) (
	backend.CreateSavedSearchResponseObject, error) {
	// At this point, the user should be authenticated and in the context.
	// If for some reason the user is not in the context, it is a library or
	// internal issue and not an user issue. Return 500 error in that case.
	user, found := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if !found {
		slog.ErrorContext(ctx, "user not found in context. middleware malfunction")

		return backend.CreateSavedSearch500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
		}, nil
	}

	validationErr := validateSavedSearch(request.Body)
	if validationErr != nil {
		return backend.CreateSavedSearch400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "input validation errors",
			Errors:  validationErr.fieldErrorMap,
		}, nil
	}

	output, err := s.wptMetricsStorer.CreateUserSavedSearch(ctx, user.ID, *request.Body)
	if err != nil {
		if errors.Is(err, backendtypes.ErrUserMaxSavedSearches) {
			return backend.CreateSavedSearch403JSONResponse{
				Code:    http.StatusForbidden,
				Message: "user has reached the maximum number of allowed saved searches",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to create user saved search", "error", err)

		return backend.CreateSavedSearch500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to create user saved search",
		}, nil
	}

	err = s.eventPublisher.PublishSearchConfigurationChanged(ctx, output, user.ID, true)
	if err != nil {
		// We should not mark this as a failure. Only log it.
		slog.WarnContext(ctx, "unable to publish search configuration changed event during create", "error", err)
	}

	return backend.CreateSavedSearch201JSONResponse(*output), nil
}

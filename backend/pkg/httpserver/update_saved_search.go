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
	"log/slog"
	"net/http"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

// validateSavedSearchUpdate handles validation when updating an existing SavedSearch.
func validateSavedSearchUpdate(input *backend.UpdateSavedSearchJSONRequestBody) *fieldValidationErrors {
	activeMasks := make(map[backend.SavedSearchUpdateRequestUpdateMask]bool)
	var invalidMasks []string

	fieldErrors := &fieldValidationErrors{fieldErrorMap: nil}

	if len(input.UpdateMask) == 0 {
		fieldErrors.addFieldError("update_mask", errors.New("update_mask must be set"))

		return fieldErrors
	}

	for _, mask := range input.UpdateMask {
		switch mask {
		case
			backend.SavedSearchUpdateRequestMaskName,
			backend.SavedSearchUpdateRequestMaskQuery,
			backend.SavedSearchUpdateRequestMaskDescription:
			activeMasks[mask] = true
		default:
			invalidMasks = append(invalidMasks, string(mask))
		}
	}

	if len(invalidMasks) > 0 {
		fieldErrors.addFieldError("update_mask", errors.New("invalid update_mask values: "+
			strings.Join(invalidMasks, ", "),
		))

		return fieldErrors
	}

	// Validate Name only if it's in the update mask
	if activeMasks[backend.SavedSearchUpdateRequestMaskName] {
		validateSavedSearchName(input.Name, fieldErrors)
	}

	// Validate Query only if it's in the update mask
	if activeMasks[backend.SavedSearchUpdateRequestMaskQuery] {
		// Original logic also checked for nil before length/parsing.
		validateSavedSearchQuery(input.Query, fieldErrors)
	}

	// Validate Description only if it's in the update mask
	if activeMasks[backend.SavedSearchUpdateRequestMaskDescription] {
		validateSavedSearchDescription(input.Description, fieldErrors)
	}

	if fieldErrors.hasErrors() {
		return fieldErrors
	}

	return nil
}

// UpdateSavedSearch implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) UpdateSavedSearch(
	ctx context.Context, request backend.UpdateSavedSearchRequestObject) (
	backend.UpdateSavedSearchResponseObject, error) {
	// At this point, the user should be authenticated and in the context.
	// If for some reason the user is not in the context, it is a library or
	// internal issue and not an user issue. Return 500 error in that case.
	user, found := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if !found {
		slog.ErrorContext(ctx, "user not found in context. middleware malfunction")

		return backend.UpdateSavedSearch500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
		}, nil
	}
	validationErr := validateSavedSearchUpdate(request.Body)
	if validationErr != nil {
		return backend.UpdateSavedSearch400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "input validation errors",
			Errors:  validationErr.fieldErrorMap,
		}, nil
	}
	output, err := s.wptMetricsStorer.UpdateUserSavedSearch(ctx, request.SearchId, user.ID, request.Body)
	if err != nil {
		if errors.Is(err, backendtypes.ErrUserNotAuthorizedForAction) {
			return backend.UpdateSavedSearch403JSONResponse{
				Code:    http.StatusForbidden,
				Message: "forbidden",
			}, nil
		} else if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.UpdateSavedSearch404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "saved search not found",
			}, nil
		}
		slog.ErrorContext(ctx, "unable to update user saved search", "error", err)

		return backend.UpdateSavedSearch500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to update user saved search",
		}, nil
	}

	return backend.UpdateSavedSearch200JSONResponse(*output), nil
}

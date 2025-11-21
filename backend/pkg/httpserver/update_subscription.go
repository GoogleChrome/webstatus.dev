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
	"net/http"
	"slices"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

var (
	errSubscriptionUpdateMaskRequired = errors.New("update_mask is required")
	errSubscriptionInvalidUpdateMask  = fmt.Errorf("update_mask must be one of the following: %s",
		getAllSubscriptionUpdateMasksToStringSlice())
)

func getAllSubscriptionUpdateMasksSet() map[backend.UpdateSubscriptionRequestUpdateMask]any {
	return map[backend.UpdateSubscriptionRequestUpdateMask]any{
		backend.UpdateSubscriptionRequestMaskTriggers:  nil,
		backend.UpdateSubscriptionRequestMaskFrequency: nil,
	}
}

func getAllSubscriptionUpdateMasksToStringSlice() []string {
	allUpdateMasks := getAllSubscriptionUpdateMasksSet()
	allUpdateMasksSlice := make([]string, 0, len(allUpdateMasks))
	for updateMask := range allUpdateMasks {
		allUpdateMasksSlice = append(allUpdateMasksSlice, string(updateMask))
	}
	slices.Sort(allUpdateMasksSlice)

	return allUpdateMasksSlice
}

func validateSubscriptionUpdateMask(updateMask *[]backend.UpdateSubscriptionRequestUpdateMask,
	required bool, fieldErrors *fieldValidationErrors) {
	if updateMask == nil {
		if required {
			fieldErrors.addFieldError("update_mask", errSubscriptionUpdateMaskRequired)
		}

		return
	}

	set := getAllSubscriptionUpdateMasksSet()
	for _, updateMask := range *updateMask {
		if _, ok := set[updateMask]; !ok {
			fieldErrors.addFieldError("update_mask", errSubscriptionInvalidUpdateMask)
		}
	}
}

func validateSubscriptionUpdate(input *backend.UpdateSubscriptionRequest) *fieldValidationErrors {
	fieldErrors := &fieldValidationErrors{fieldErrorMap: nil}

	validateSubscriptionUpdateMask(&input.UpdateMask, true, fieldErrors)

	isTriggerRequired := slices.Contains(input.UpdateMask, backend.UpdateSubscriptionRequestMaskTriggers)
	validateSubscriptionTrigger(input.Triggers, isTriggerRequired, fieldErrors)

	isFrequencyRequired := slices.Contains(input.UpdateMask, backend.UpdateSubscriptionRequestMaskFrequency)
	validateSubscriptionFrequency(input.Frequency, isFrequencyRequired, fieldErrors)

	if fieldErrors.hasErrors() {
		return fieldErrors
	}

	return nil
}

// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) UpdateSubscription(
	ctx context.Context,
	request backend.UpdateSubscriptionRequestObject,
) (backend.UpdateSubscriptionResponseObject, error) {
	userCheck := CheckAuthenticatedUser[backend.UpdateSubscriptionResponseObject](ctx, "UpdateSubscription",
		func(code int, message string) backend.UpdateSubscriptionResponseObject {
			return backend.UpdateSubscription500JSONResponse(backend.BasicErrorModel{Code: code, Message: message})
		})
	if userCheck.User == nil {
		return userCheck.Response, nil
	}

	validationErr := validateSubscriptionUpdate(request.Body)
	if validationErr != nil {
		return backend.UpdateSubscription400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "input validation errors",
			Errors:  validationErr.fieldErrorMap,
		}, nil
	}

	resp, err := s.wptMetricsStorer.UpdateSavedSearchSubscription(
		ctx, userCheck.User.ID, request.SubscriptionId, *request.Body)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.UpdateSubscription404JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusNotFound,
					Message: "subscription not found",
				},
			), nil
		}

		return nil, err
	}

	return backend.UpdateSubscription200JSONResponse(*resp), nil
}

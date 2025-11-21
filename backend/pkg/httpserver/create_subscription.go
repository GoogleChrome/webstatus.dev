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

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// The exhaustive linter is configured to check that this map is complete.
func getAllSubscriptionTriggersSet() map[backend.SubscriptionTrigger]any {
	return map[backend.SubscriptionTrigger]any{
		backend.SubscriptionTriggerFeatureAnyBrowserImplementationComplete: nil,
		backend.SubscriptionTriggerFeatureBaselineLimitedToNewly:           nil,
		backend.SubscriptionTriggerFeatureBaselineRegressionNewlyToLimited: nil,
	}
}

func getAllSubscriptionTriggersToStringSlice() []string {
	allTriggers := getAllSubscriptionTriggersSet()
	allTriggersSlice := make([]string, 0, len(allTriggers))
	for trigger := range allTriggers {
		allTriggersSlice = append(allTriggersSlice, string(trigger))
	}
	slices.Sort(allTriggersSlice)

	return allTriggersSlice
}

var (
	errSubscriptionInvalidTrigger = fmt.Errorf("triggers must be one of the following: %s",
		getAllSubscriptionTriggersToStringSlice())
	errSubscriptionInvalidFrequency = fmt.Errorf("frequency must be one of the following: %s",
		getAllSubscriptionFrequenciesToStringSlice())
	errSubscriptionChannelIDRequired     = errors.New("channel_id is required")
	errSubscriptionSavedSearchIDRequired = errors.New("saved_search_id is required")
)

func validateSubscriptionTrigger(trigger *[]backend.SubscriptionTrigger,
	required bool, fieldErrors *fieldValidationErrors) {
	if trigger == nil {
		if required {
			fieldErrors.addFieldError("triggers", errSubscriptionInvalidTrigger)
		}

		return
	}

	set := getAllSubscriptionTriggersSet()
	for _, trigger := range *trigger {
		if _, ok := set[trigger]; !ok {
			fieldErrors.addFieldError("triggers", errSubscriptionInvalidTrigger)
		}
	}
}

// The exhaustive linter is configured to check that this map is complete.
func getAllSubscriptionFrequenciesSet() map[backend.SubscriptionFrequency]any {
	return map[backend.SubscriptionFrequency]any{
		backend.SubscriptionFrequencyDaily: nil,
	}
}

func getAllSubscriptionFrequenciesToStringSlice() []string {
	allFrequencies := getAllSubscriptionFrequenciesSet()
	allFrequenciesSlice := make([]string, 0, len(allFrequencies))
	for frequency := range allFrequencies {
		allFrequenciesSlice = append(allFrequenciesSlice, string(frequency))
	}
	slices.Sort(allFrequenciesSlice)

	return allFrequenciesSlice
}

func validateSubscriptionFrequency(frequency *backend.SubscriptionFrequency,
	required bool, fieldErrors *fieldValidationErrors) {
	if frequency == nil {
		if required {
			fieldErrors.addFieldError("frequency", errSubscriptionInvalidFrequency)
		}

		return
	}

	set := getAllSubscriptionFrequenciesSet()
	if _, ok := set[*frequency]; !ok {
		fieldErrors.addFieldError("frequency", errSubscriptionInvalidFrequency)
	}
}

func validateSubscriptionChannelID(channelID string, fieldErrors *fieldValidationErrors) {
	if channelID == "" {
		fieldErrors.addFieldError("channel_id", errSubscriptionChannelIDRequired)

		return
	}
}

func validateSubscriptionSavedSearchID(savedSearchID string, fieldErrors *fieldValidationErrors) {
	if savedSearchID == "" {
		fieldErrors.addFieldError("saved_search_id", errSubscriptionSavedSearchIDRequired)

		return
	}
}

func validateSubscriptionCreation(input *backend.Subscription) *fieldValidationErrors {
	fieldErrors := &fieldValidationErrors{fieldErrorMap: nil}

	validateSubscriptionTrigger(&input.Triggers, true, fieldErrors)

	validateSubscriptionFrequency(&input.Frequency, true, fieldErrors)

	validateSubscriptionChannelID(input.ChannelId, fieldErrors)

	validateSubscriptionSavedSearchID(input.SavedSearchId, fieldErrors)

	if fieldErrors.hasErrors() {
		return fieldErrors
	}

	return nil
}

// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) CreateSubscription(
	ctx context.Context,
	request backend.CreateSubscriptionRequestObject,
) (backend.CreateSubscriptionResponseObject, error) {
	userCheck := CheckAuthenticatedUser[backend.CreateSubscriptionResponseObject](ctx, "CreateSubscription",
		func(code int, message string) backend.CreateSubscriptionResponseObject {
			return backend.CreateSubscription500JSONResponse(backend.BasicErrorModel{Code: code, Message: message})
		})
	if userCheck.User == nil {
		return userCheck.Response, nil
	}
	validationErr := validateSubscriptionCreation(request.Body)
	if validationErr != nil {
		return backend.CreateSubscription400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "input validation errors",
			Errors:  validationErr.fieldErrorMap,
		}, nil
	}

	resp, err := s.wptMetricsStorer.CreateSavedSearchSubscription(ctx, userCheck.User.ID, *request.Body)
	if err != nil {
		return nil, err
	}

	return backend.CreateSubscription201JSONResponse(*resp), nil
}

// Copyright 2026 Google LLC
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

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

func validateNotificationChannel(input *backend.CreateNotificationChannelRequest) *fieldValidationErrors {
	fieldErrors := &fieldValidationErrors{fieldErrorMap: nil}

	if len(input.Name) < notificationChannelNameMinLength || len(input.Name) > notificationChannelNameMaxLength {
		fieldErrors.addFieldError("name", errNotificationChannelInvalidNameLength)
	}

	if cfg, err := input.Config.AsWebhookConfig(); err == nil && cfg.Type == backend.WebhookConfigTypeWebhook {
		if err := validateSlackWebhookURL(cfg.Url); err != nil {
			fieldErrors.addFieldError("config.url", err)
		}
	} else {
		fieldErrors.addFieldError("config", errors.New("invalid config: only webhook channels can be created manually"))
	}

	if fieldErrors.hasErrors() {
		return fieldErrors
	}

	return nil
}

// CreateNotificationChannel implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) CreateNotificationChannel(
	ctx context.Context,
	request backend.CreateNotificationChannelRequestObject) (
	backend.CreateNotificationChannelResponseObject, error) {
	user, found := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if !found {
		slog.ErrorContext(ctx, "user not found in context. middleware malfunction")

		return backend.CreateNotificationChannel500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
		}, nil
	}

	validationErr := validateNotificationChannel(request.Body)
	if validationErr != nil {
		return backend.CreateNotificationChannel400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "input validation errors",
			Errors:  validationErr.fieldErrorMap,
		}, nil
	}

	output, err := s.wptMetricsStorer.CreateNotificationChannel(ctx, user.ID, *request.Body)
	if err != nil {
		if errors.Is(err, backendtypes.ErrUserMaxNotificationChannels) {
			return backend.CreateNotificationChannel429JSONResponse{
				Code:    http.StatusTooManyRequests,
				Message: "user has reached the maximum number of allowed notification channels (25)",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to create notification channel", "error", err)

		return backend.CreateNotificationChannel500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to create notification channel",
		}, nil
	}

	return backend.CreateNotificationChannel201JSONResponse(*output), nil
}

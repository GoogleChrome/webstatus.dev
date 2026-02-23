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

func validateUpdateNotificationChannel(request *backend.UpdateNotificationChannelRequest) *fieldValidationErrors {
	fieldErrors := &fieldValidationErrors{fieldErrorMap: nil}

	for _, mask := range request.UpdateMask {
		switch mask {
		case backend.UpdateNotificationChannelRequestMaskName:
			if request.Name == nil ||
				len(*request.Name) < notificationChannelNameMinLength ||
				len(*request.Name) > notificationChannelNameMaxLength {
				fieldErrors.addFieldError("name", errNotificationChannelInvalidNameLength)
			}
		case backend.UpdateNotificationChannelRequestMaskConfig:
			if request.Config == nil {
				fieldErrors.addFieldError("config", errors.New("config must be set"))

				continue
			}

			if cfg, err := request.Config.AsWebhookConfig(); err == nil && cfg.Type == backend.WebhookConfigTypeWebhook {
				if err := validateSlackWebhookURL(cfg.Url); err != nil {
					fieldErrors.addFieldError("config.url", err)
				}
			} else {
				fieldErrors.addFieldError("config", errors.New("invalid config: only webhook updates are supported"))
			}
		}
	}

	if fieldErrors.hasErrors() {
		return fieldErrors
	}

	return nil
}

// UpdateNotificationChannel implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) UpdateNotificationChannel(
	ctx context.Context,
	request backend.UpdateNotificationChannelRequestObject) (
	backend.UpdateNotificationChannelResponseObject, error) {
	// At this point, the user should be authenticated and in the context.
	// If for some reason the user is not in the context, it is a library or
	// internal issue and not an user issue. Return 500 error in that case.
	user, found := httpmiddlewares.AuthenticatedUserFromContext(ctx)
	if !found {
		slog.ErrorContext(ctx, "user not found in context. middleware malfunction")

		return backend.UpdateNotificationChannel500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
		}, nil
	}

	validationErr := validateUpdateNotificationChannel(request.Body)
	if validationErr != nil {
		return backend.UpdateNotificationChannel400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "input validation errors",
			Errors:  validationErr.fieldErrorMap,
		}, nil
	}

	output, err := s.wptMetricsStorer.UpdateNotificationChannel(ctx, user.ID, request.ChannelId, *request.Body)
	if err != nil {
		if errors.Is(err, backendtypes.ErrUserNotAuthorizedForAction) {
			return backend.UpdateNotificationChannel403JSONResponse{
				Code:    http.StatusForbidden,
				Message: "email notification channels cannot be updated manually",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to update notification channel", "error", err, "channelID", request.ChannelId)

		return backend.UpdateNotificationChannel500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "unable to update notification channel",
		}, nil
	}

	return backend.UpdateNotificationChannel200JSONResponse(*output), nil
}

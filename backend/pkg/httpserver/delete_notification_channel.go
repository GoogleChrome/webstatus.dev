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
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// DeleteNotificationChannel handles the DELETE request to /v1/users/me/notification-channels/{channel_id}.
// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) DeleteNotificationChannel(
	ctx context.Context,
	req backend.DeleteNotificationChannelRequestObject,
) (backend.DeleteNotificationChannelResponseObject, error) {
	userCheckResult := CheckAuthenticatedUser(ctx, "DeleteNotificationChannel",
		func(code int, message string) backend.DeleteNotificationChannel500JSONResponse {
			return backend.DeleteNotificationChannel500JSONResponse{
				Code:    code,
				Message: message,
			}
		})
	if userCheckResult.User == nil {
		return userCheckResult.Response, nil
	}

	err := s.wptMetricsStorer.DeleteNotificationChannel(ctx, userCheckResult.User.ID, req.ChannelId)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) || errors.Is(err, backendtypes.ErrUserNotAuthorizedForAction) {
			return backend.DeleteNotificationChannel404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "Notification channel not found or not owned by user",
			}, nil
		}

		return backend.DeleteNotificationChannel500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "Could not delete notification channel",
		}, nil
	}

	return backend.DeleteNotificationChannel204Response{}, nil
}

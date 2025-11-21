// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by a applicable law or agreed to in writing, software
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

// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) DeleteSubscription(
	ctx context.Context,
	request backend.DeleteSubscriptionRequestObject,
) (backend.DeleteSubscriptionResponseObject, error) {
	userCheck := CheckAuthenticatedUser[backend.DeleteSubscriptionResponseObject](ctx, "DeleteSubscription",
		func(code int, message string) backend.DeleteSubscriptionResponseObject {
			return backend.DeleteSubscription500JSONResponse(backend.BasicErrorModel{Code: code, Message: message})
		})
	if userCheck.User == nil {
		return userCheck.Response, nil
	}

	err := s.wptMetricsStorer.DeleteSavedSearchSubscription(ctx, userCheck.User.ID, request.SubscriptionId)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.DeleteSubscription404JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusNotFound,
					Message: "subscription not found",
				},
			), nil
		}

		return nil, err
	}

	return backend.DeleteSubscription204Response{}, nil
}

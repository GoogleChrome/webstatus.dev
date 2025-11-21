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

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) ListSubscriptions(
	ctx context.Context,
	request backend.ListSubscriptionsRequestObject,
) (backend.ListSubscriptionsResponseObject, error) {
	userCheck := CheckAuthenticatedUser[backend.ListSubscriptionsResponseObject](ctx, "ListSubscriptions",
		func(code int, message string) backend.ListSubscriptionsResponseObject {
			return backend.ListSubscriptions500JSONResponse(backend.BasicErrorModel{Code: code, Message: message})
		})
	if userCheck.User == nil {
		return userCheck.Response, nil
	}

	resp, err := s.wptMetricsStorer.ListSavedSearchSubscriptions(
		ctx, userCheck.User.ID, getPageSizeOrDefault(request.Params.PageSize), request.Params.PageToken)
	if err != nil {
		return nil, err
	}

	return backend.ListSubscriptions200JSONResponse(*resp), nil
}

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

// PingUser implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) PingUser(
	ctx context.Context,
	_ backend.PingUserRequestObject,
) (backend.PingUserResponseObject, error) {
	userCheckResult := CheckAuthenticatedUser(ctx, "PingUser",
		func(code int, message string) backend.PingUser500JSONResponse {
			return backend.PingUser500JSONResponse{
				Code:    code,
				Message: message,
			}
		})
	if userCheckResult.User == nil {
		return userCheckResult.Response, nil
	}

	// TODO: Implement database logic to upsert user profile.
	return backend.PingUser204Response{}, nil
}

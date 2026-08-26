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

//nolint:dupl // Similar structure across listing handlers
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListVCSRepositories handles GET /v1/vcs/{provider}/repositories.
//
//nolint:ireturn, revive, dupl // Expected ireturn for openapi generation.
func (s *Server) ListVCSRepositories(
	ctx context.Context,
	request backend.ListVCSRepositoriesRequestObject,
) (backend.ListVCSRepositoriesResponseObject, error) {
	userCheck := CheckAuthenticatedUser[backend.ListVCSRepositoriesResponseObject](
		ctx, "ListVCSRepositories",
		func(code int, message string) backend.ListVCSRepositoriesResponseObject {
			return backend.ListVCSRepositories500JSONResponse(
				backend.BasicErrorModel{Code: code, Message: message})
		})
	if userCheck.User == nil {
		return userCheck.Response, nil
	}

	pageSize := getPageSizeOrDefault(request.Params.PageSize)

	page, err := s.wptMetricsStorer.ListVCSRepositories(
		ctx, request.Provider, pageSize, request.Params.PageToken)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			return backend.ListVCSRepositories400JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusBadRequest,
					Message: errMsgInvalidPageToken,
				}), nil
		}
		if errors.Is(err, backendtypes.ErrUnsupportedVCSProvider) {
			return backend.ListVCSRepositories400JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusBadRequest,
					Message: fmt.Sprintf("unsupported VCS provider: %s", request.Provider),
				}), nil
		}

		slog.ErrorContext(ctx, "failed to list VCS repositories", "error", err,
			"provider", request.Provider)

		return backend.ListVCSRepositories500JSONResponse(
			backend.BasicErrorModel{
				Code:    http.StatusInternalServerError,
				Message: "failed to list VCS repositories",
			}), nil
	}

	return backend.ListVCSRepositories200JSONResponse(*page), nil
}

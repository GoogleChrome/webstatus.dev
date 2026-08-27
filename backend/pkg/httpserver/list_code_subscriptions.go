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
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListCodeSubscriptions handles GET /v1/vcs/{provider}/repositories/{repository_id}/code-subscriptions.
//
//nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) ListCodeSubscriptions(
	ctx context.Context,
	request backend.ListCodeSubscriptionsRequestObject,
) (backend.ListCodeSubscriptionsResponseObject, error) {
	userCheck := CheckAuthenticatedUser[backend.ListCodeSubscriptionsResponseObject](
		ctx, "ListCodeSubscriptions",
		func(code int, message string) backend.ListCodeSubscriptionsResponseObject {
			return backend.ListCodeSubscriptions500JSONResponse(
				backend.BasicErrorModel{Code: code, Message: message})
		})
	if userCheck.User == nil {
		return userCheck.Response, nil
	}

	if request.Provider != "github" {
		return backend.ListCodeSubscriptions400JSONResponse(
			backend.BasicErrorModel{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("unsupported VCS provider: %s", request.Provider),
			}), nil
	}

	owner, repo := "", request.RepositoryId
	if parts := strings.Split(request.RepositoryId, "/"); len(parts) == 2 {
		owner, repo = parts[0], parts[1]
	}

	if userCheck.User.GitHubUserID == nil || *userCheck.User.GitHubUserID == "" {
		return backend.ListCodeSubscriptions404JSONResponse(
			backend.BasicErrorModel{
				Code:    http.StatusNotFound,
				Message: errMsgRepositoryNotFound,
			}), nil
	}

	if s.vcsPermissionChecker != nil {
		hasAdminAccess, err := s.vcsPermissionChecker.HasRepositoryAdminAccess(
			ctx, owner, repo, *userCheck.User.GitHubUserID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check repository permissions", "error", err,
				"provider", request.Provider, "repo_id", request.RepositoryId)

			return backend.ListCodeSubscriptions500JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusInternalServerError,
					Message: "failed to check repository permissions",
				}), nil
		}
		if !hasAdminAccess {
			return backend.ListCodeSubscriptions404JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusNotFound,
					Message: errMsgRepositoryNotFound,
				}), nil
		}
	}

	pageSize := getPageSizeOrDefault(request.Params.PageSize)

	page, err := s.wptMetricsStorer.ListCodeSubscriptions(
		ctx, request.Provider, request.RepositoryId, pageSize, request.Params.PageToken)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			return backend.ListCodeSubscriptions400JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusBadRequest,
					Message: errMsgInvalidPageToken,
				}), nil
		}
		if errors.Is(err, backendtypes.ErrUnsupportedVCSProvider) {
			return backend.ListCodeSubscriptions400JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusBadRequest,
					Message: fmt.Sprintf("unsupported VCS provider: %s", request.Provider),
				}), nil
		}
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.ListCodeSubscriptions404JSONResponse(
				backend.BasicErrorModel{
					Code:    http.StatusNotFound,
					Message: errMsgRepositoryNotFound,
				}), nil
		}

		slog.ErrorContext(ctx, "failed to list code subscriptions", "error", err,
			"provider", request.Provider, "repo_id", request.RepositoryId)

		return backend.ListCodeSubscriptions500JSONResponse(
			backend.BasicErrorModel{
				Code:    http.StatusInternalServerError,
				Message: "failed to list code subscriptions",
			}), nil
	}

	return backend.ListCodeSubscriptions200JSONResponse(*page), nil
}

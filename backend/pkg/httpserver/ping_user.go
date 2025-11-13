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
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// PingUser implements backend.StrictServerInterface.
// nolint: ireturn // Name generated from openapi
func (s *Server) PingUser(
	ctx context.Context,
	req backend.PingUserRequestObject,
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

	user := userCheckResult.User
	if req.Body.GithubToken == nil {
		// Nothing to do.

		return backend.PingUser204Response{}, nil
	}

	if user.GitHubUserID == nil {
		slog.ErrorContext(ctx, "token is missing github user id", "user", user.ID)

		return backend.PingUser500JSONResponse{
			Code:    500,
			Message: "token is missing github user id",
		}, nil
	}

	githubClient := s.userGitHubClientFactory(*req.Body.GithubToken)
	githubUser, err := githubClient.GetCurrentUser(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get GitHub user", "error", err, "user", user.ID)

		return backend.PingUser500JSONResponse{
			Code:    500,
			Message: "failed to get GitHub user",
		}, nil
	}

	if !user.HasGitHubUserID(githubUser.ID) {
		slog.WarnContext(ctx, "user does not have specified GitHub User ID", "user", user.ID,
			"github_username", githubUser.Username)

		return backend.PingUser403JSONResponse{
			Code:    403,
			Message: "user does not match specified GitHub User ID",
		}, nil
	}

	emails, err := githubClient.ListEmails(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list GitHub emails", "error", err, "user", user.ID,
			"github ID", githubUser.ID)

		return backend.PingUser500JSONResponse{
			Code:    500,
			Message: "failed to list GitHub emails",
		}, nil
	}
	var verifiedEmails []string
	for _, email := range emails {
		if email.Verified {
			verifiedEmails = append(verifiedEmails, email.Email)
		}
	}

	err = s.wptMetricsStorer.SyncUserProfileInfo(ctx, backendtypes.UserProfile{
		UserID:       user.ID,
		GitHubUserID: githubUser.ID,
		Emails:       verifiedEmails,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to sync user profile", "error", err, "user", user.ID)

		return backend.PingUser500JSONResponse{
			Code:    500,
			Message: "failed to sync user profile",
		}, nil
	}

	return backend.PingUser204Response{}, nil
}

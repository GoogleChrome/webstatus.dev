// Copyright 2024 Google LLC
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

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListMissingOneImplementationFeatures implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListMissingOneImplementationFeatures(
	ctx context.Context,
	request backend.ListMissingOneImplementationFeaturesRequestObject) (
	backend.ListMissingOneImplementationFeaturesResponseObject, error) {
	otherBrowsers := make([]string, len(request.Params.Browser))
	for i := 0; i < len(request.Params.Browser); i++ {
		otherBrowsers[i] = string(request.Params.Browser[i])
	}
	page, err := s.wptMetricsStorer.ListMissingOneImplementationFeatures(
		ctx,
		string(request.Browser),
		otherBrowsers,
		request.Date.Time,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", request.Params.PageToken, "error", err)

			return backend.ListMissingOneImplementationFeatures400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get missing one implementation feature list", "error", err)

		return backend.ListMissingOneImplementationFeatures500JSONResponse{
			Code:    500,
			Message: "unable to get missing one implementation feature list",
		}, nil
	}

	resp := backend.ListMissingOneImplementationFeatures200JSONResponse{
		Metadata: page.Metadata,
		Data:     page.Data,
	}

	return resp, nil
}

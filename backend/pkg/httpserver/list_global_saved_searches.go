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

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListGlobalSavedSearches implements backend.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) ListGlobalSavedSearches(
	ctx context.Context,
	req backend.ListGlobalSavedSearchesRequestObject,
) (backend.ListGlobalSavedSearchesResponseObject, error) {
	var cachedResponse backend.ListGlobalSavedSearches200JSONResponse
	found := s.operationResponseCaches.listGlobalSavedSearchesCache.Lookup(ctx, req, &cachedResponse)
	if found {
		return cachedResponse, nil
	}

	page, err := s.wptMetricsStorer.ListGlobalSavedSearches(
		ctx,
		getPageSizeOrDefault(req.Params.PageSize),
		req.Params.PageToken,
	)

	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", req.Params.PageToken, "error", err)

			return backend.ListGlobalSavedSearches400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get list of global saved searches", "error", err)

		return backend.ListGlobalSavedSearches500JSONResponse{
			Code:    500,
			Message: "unable to get list of global saved searches",
		}, nil
	}

	resp := backend.ListGlobalSavedSearches200JSONResponse{
		Metadata: page.Metadata,
		Data:     page.Data,
	}
	s.operationResponseCaches.listGlobalSavedSearchesCache.AttemptCache(ctx, req, &resp)

	return resp, nil
}

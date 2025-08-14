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

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListMissingOneImplementationCounts implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListMissingOneImplementationCounts(
	ctx context.Context,
	request backend.ListMissingOneImplementationCountsRequestObject) (
	backend.ListMissingOneImplementationCountsResponseObject, error) {
	var cachedResponse backend.ListMissingOneImplementationCounts200JSONResponse
	found := s.operationResponseCaches.ListMissingOneImplementationCountsCache.Lookup(ctx, request, &cachedResponse)
	if found {
		return cachedResponse, nil
	}

	browserParams, err := PrepareMissingOneBrowserParams(
		request.Browser, request.Params.Browser, request.Params.IncludeBaselineMobileBrowsers != nil)
	if err != nil {
		if errors.Is(err, ErrNoMatchingMobileBrowser) {
			return backend.ListMissingOneImplementationCounts400JSONResponse{
				Code:    400,
				Message: err.Error(),
			}, nil
		}

		return backend.ListMissingOneImplementationCounts500JSONResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	page, err := s.wptMetricsStorer.ListMissingOneImplCounts(
		ctx,
		browserParams.targetBrowser,
		browserParams.targetMobileBrowser,
		browserParams.otherBrowsers,
		request.Params.StartAt.Time,
		request.Params.EndAt.Time,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", request.Params.PageToken, "error", err)

			return backend.ListMissingOneImplementationCounts400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get missing one implementation count", "error", err)

		return backend.ListMissingOneImplementationCounts500JSONResponse{
			Code:    500,
			Message: "unable to get missing one implementation metrics",
		}, nil
	}

	resp := backend.ListMissingOneImplementationCounts200JSONResponse{
		Metadata: page.Metadata,
		Data:     page.Data,
	}
	s.operationResponseCaches.ListMissingOneImplementationCountsCache.AttemptCache(ctx, request, &resp)

	return resp, nil
}

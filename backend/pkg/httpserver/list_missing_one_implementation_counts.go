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

// ListMissingOneImplemenationCounts implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListMissingOneImplemenationCounts(
	ctx context.Context,
	request backend.ListMissingOneImplemenationCountsRequestObject) (
	backend.ListMissingOneImplemenationCountsResponseObject, error) {
	var cachedResponse backend.ListMissingOneImplemenationCounts200JSONResponse
	found := s.operationResponseCaches.listMissingOneImplemenationCountsCache.Lookup(ctx, request, &cachedResponse)
	if found {
		return cachedResponse, nil
	}
	otherBrowsers := make([]string, len(request.Params.Browser))
	for i := 0; i < len(request.Params.Browser); i++ {
		otherBrowsers[i] = string(request.Params.Browser[i])
	}
	page, err := s.wptMetricsStorer.ListMissingOneImplCounts(
		ctx,
		string(request.Browser),
		otherBrowsers,
		request.Params.StartAt.Time,
		request.Params.EndAt.Time,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", request.Params.PageToken, "error", err)

			return backend.ListMissingOneImplemenationCounts400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get missing one implementation count", "error", err)

		return backend.ListMissingOneImplemenationCounts500JSONResponse{
			Code:    500,
			Message: "unable to get missing one implementation metrics",
		}, nil
	}

	resp := backend.ListMissingOneImplemenationCounts200JSONResponse{
		Metadata: page.Metadata,
		Data:     page.Data,
	}
	s.operationResponseCaches.listMissingOneImplemenationCountsCache.AttemptCache(ctx, request, &resp)

	return resp, nil
}

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

// ListChromiumDailyUsageStats implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) ListChromiumDailyUsageStats(
	ctx context.Context,
	request backend.ListChromiumDailyUsageStatsRequestObject,
) (backend.ListChromiumDailyUsageStatsResponseObject, error) {
	var cachedResponse backend.ListChromiumDailyUsageStats200JSONResponse
	found := s.operationResponseCaches.listChromiumDailyUsageStatsCache.Lookup(ctx, request, &cachedResponse)
	if found {
		return cachedResponse, nil
	}
	stats, nextPageToken, err := s.wptMetricsStorer.ListChromiumDailyUsageStats(
		ctx,
		request.FeatureId,
		request.Params.StartAt.Time,
		request.Params.EndAt.Time,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", request.Params.PageToken, "error", err)

			return backend.ListChromiumDailyUsageStats400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get feature metrics", "error", err)

		return backend.ListChromiumDailyUsageStats500JSONResponse{
			Code:    500,
			Message: "unable to get feature metrics",
		}, nil
	}

	resp := backend.ListChromiumDailyUsageStats200JSONResponse{
		Data: stats,
		Metadata: &backend.PageMetadata{
			NextPageToken: nextPageToken,
		},
	}
	s.operationResponseCaches.listChromiumDailyUsageStatsCache.AttemptCache(ctx, request, &resp)

	return resp, nil
}

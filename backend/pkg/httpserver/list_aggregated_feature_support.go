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

// ListAggregatedFeatureSupport implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListAggregatedFeatureSupport(
	ctx context.Context,
	request backend.ListAggregatedFeatureSupportRequestObject) (
	backend.ListAggregatedFeatureSupportResponseObject, error) {
	page, err := s.wptMetricsStorer.ListBrowserFeatureCountMetric(
		ctx,
		string(request.Browser),
		request.Params.StartAt.Time,
		request.Params.EndAt.Time,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", request.Params.PageToken, "error", err)

			return backend.ListAggregatedFeatureSupport400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get count of supported features", "error", err)

		return backend.ListAggregatedFeatureSupport500JSONResponse{
			Code:    500,
			Message: "unable to get feature support metrics",
		}, nil
	}

	return backend.ListAggregatedFeatureSupport200JSONResponse{
		Metadata: page.Metadata,
		Data:     page.Data,
	}, nil
}

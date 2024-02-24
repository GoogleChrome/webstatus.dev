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
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// ListAggregatedWPTMetrics implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) ListAggregatedWPTMetrics(
	ctx context.Context,
	request backend.ListAggregatedWPTMetricsRequestObject,
) (backend.ListAggregatedWPTMetricsResponseObject, error) {
	slog.Info("parameters", "feature", getFeatureIDsOrDefault(request.Params.FeatureIds))

	metrics, nextPageToken, err := s.wptMetricsStorer.ListMetricsOverTimeWithAggregatedTotals(
		ctx,
		getFeatureIDsOrDefault(request.Params.FeatureIds),
		string(request.Browser),
		string(request.Channel),
		request.Params.StartAt.Time,
		request.Params.EndAt.Time,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)
	if err != nil {
		slog.Error("unable to get aggregated metrics", "error", err)

		return backend.ListAggregatedWPTMetrics500JSONResponse{
			Code:    500,
			Message: "unable to get aggregated metrics",
		}, nil
	}

	return backend.ListAggregatedWPTMetrics200JSONResponse{
		Data: metrics,
		Metadata: &backend.PageMetadata{
			NextPageToken: nextPageToken,
		},
	}, nil
}

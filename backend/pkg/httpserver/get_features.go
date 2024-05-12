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
	"net/http"
	"net/url"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/searchtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// GetV1Features implements backend.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) GetV1Features(
	ctx context.Context,
	req backend.GetV1FeaturesRequestObject,
) (backend.GetV1FeaturesResponseObject, error) {
	var node *searchtypes.SearchNode
	if req.Params.Q != nil {
		// Try to decode the url.
		decodedStr, err := url.QueryUnescape(*req.Params.Q)
		if err != nil {
			slog.WarnContext(ctx, "unable to decode string", "input string", *req.Params.Q, "error", err)

			return backend.GetV1Features400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: "query string cannot be decoded",
			}, nil
		}

		parser := searchtypes.FeaturesSearchQueryParser{}
		node, err = parser.Parse(decodedStr)
		if err != nil {
			slog.WarnContext(ctx, "unable to parse query string", "query", decodedStr, "error", err)

			return backend.GetV1Features400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: "query string does not match expected grammar",
			}, nil
		}
	}
	featurePage, err := s.wptMetricsStorer.FeaturesSearch(
		ctx,
		req.Params.PageToken,
		getPageSizeOrDefault(req.Params.PageSize),
		node,
		req.Params.Sort,
		getWPTMetricViewOrDefault(req.Params.WptMetricView),
		defaultBrowsers(),
	)

	if err != nil {
		// TODO check error type
		slog.ErrorContext(ctx, "unable to get list of features", "error", err)

		return backend.GetV1Features500JSONResponse{
			Code:    500,
			Message: "unable to get list of features",
		}, nil
	}

	return backend.GetV1Features200JSONResponse{
		Metadata: featurePage.Metadata,
		Data:     featurePage.Data,
	}, nil
}

func getWPTMetricViewOrDefault(in *backend.WPTMetricView) backend.WPTMetricView {
	if in != nil {
		switch *in {
		case backend.SubtestCounts, backend.TestCounts:
			return *in
		}
	}

	// Default to subtest count if not specified or invalid metric view.
	return backend.SubtestCounts
}

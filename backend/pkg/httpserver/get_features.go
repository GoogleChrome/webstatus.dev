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

// GetV1Features implements backend.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) GetV1Features(
	ctx context.Context,
	req backend.GetV1FeaturesRequestObject,
) (backend.GetV1FeaturesResponseObject, error) {
	featureData, nextPageToken, err := s.wptMetricsStorer.FeaturesSearch(
		ctx,
		req.Params.PageToken,
		getPageSizeOrDefault(req.Params.PageSize),
		getBrowserListOrDefault(req.Params.AvailableOn),
		getBrowserListOrDefault(req.Params.NotAvailableOn),
	)

	if err != nil {
		// TODO check error type
		slog.Error("unable to get list of features", "error", err)

		return backend.GetV1Features500JSONResponse{
			Code:    500,
			Message: "unable to get list of features",
		}, nil
	}

	return backend.GetV1Features200JSONResponse{
		Metadata: &backend.PageMetadata{
			NextPageToken: nextPageToken,
		},
		Data: featureData,
	}, nil
}

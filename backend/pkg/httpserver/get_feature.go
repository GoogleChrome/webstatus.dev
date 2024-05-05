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
	"fmt"
	"log/slog"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

// GetV1FeaturesFeatureId implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) GetV1FeaturesFeatureId(
	ctx context.Context,
	request backend.GetV1FeaturesFeatureIdRequestObject,
) (backend.GetV1FeaturesFeatureIdResponseObject, error) {
	feature, err := s.wptMetricsStorer.GetFeature(ctx, request.FeatureId,
		getWPTMetricViewOrDefault(request.Params.WptMetricView),
		defaultBrowsers(),
	)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			return backend.GetV1FeaturesFeatureId404JSONResponse{
				Code:    http.StatusNotFound,
				Message: fmt.Sprintf("feature id %s is not found", request.FeatureId),
			}, nil
		}
		// Catch all for all other errors.
		slog.Error("unable to get feature", "error", err)

		return backend.GetV1FeaturesFeatureId500JSONResponse{
			Code:    500,
			Message: "unable to get feature",
		}, nil
	}

	return backend.GetV1FeaturesFeatureId200JSONResponse(*feature), nil
}

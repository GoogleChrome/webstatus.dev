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

// GetFeatureMetadata implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) GetFeatureMetadata(ctx context.Context,
	request backend.GetFeatureMetadataRequestObject) (backend.GetFeatureMetadataResponseObject, error) {
	featureId, err := s.wptMetricsStorer.GetIDFromFeatureKey(ctx, request.FeatureId)
	if err != nil {
		if errors.Is(err, gcpspanner.ErrQueryReturnedNoResults) {
			return backend.GetFeatureMetadata404JSONResponse{
				Code:    http.StatusNotFound,
				Message: fmt.Sprintf("feature id %s is not found", request.FeatureId),
			}, nil
		}
		// Catch all for all other errors.
		slog.Error("unable to check feature before fetching metadata", "error", err)

		return backend.GetFeatureMetadata500JSONResponse{
			Code:    500,
			Message: "unable to get feature metadata",
		}, nil
	}

	metadata, err := s.metadataStorer.GetFeatureMetadata(ctx, *featureId)
	if err != nil {
		// Catch all for all other errors.
		slog.Error("unable to get feature metadata", "error", err)

		return backend.GetFeatureMetadata500JSONResponse{
			Code:    500,
			Message: "unable to get feature metadata",
		}, nil
	}

	return backend.GetFeatureMetadata200JSONResponse(*metadata), nil
}

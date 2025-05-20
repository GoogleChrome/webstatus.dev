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

var ErrNoMatchingMobileBrowser = errors.New("browser does not have a matching mobile browser")

func getDesktopsMobileProduct(browser backend.BrowserPathParam) (*string, error) {

	var mobileBrowser string
	switch browser {
	case backend.Chrome:
		mobileBrowser = string(backend.ChromeAndroid)
		return &mobileBrowser, nil
	case backend.Firefox:
		mobileBrowser = string(backend.FirefoxAndroid)
		return &mobileBrowser, nil
	case backend.Safari:
		mobileBrowser = string(backend.SafariIos)
		return &mobileBrowser, nil
	case backend.Edge, backend.ChromeAndroid, backend.FirefoxAndroid, backend.SafariIos:
		return nil, ErrNoMatchingMobileBrowser
	}

	return &mobileBrowser, ErrNoMatchingMobileBrowser
}

// ListAggregatedFeatureSupport implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListAggregatedFeatureSupport(
	ctx context.Context,
	request backend.ListAggregatedFeatureSupportRequestObject) (
	backend.ListAggregatedFeatureSupportResponseObject, error) {
	var cachedResponse backend.ListAggregatedFeatureSupport200JSONResponse
	found := s.operationResponseCaches.listAggregatedFeatureSupportCache.Lookup(ctx, request, &cachedResponse)
	if found {
		return cachedResponse, nil
	}

	var targetMobileBrowser *string
	if request.Params.IncludeBaselineMobileBrowsers != nil {
		var err error
		targetMobileBrowser, err = getDesktopsMobileProduct(request.Browser)
		if err != nil {
			return backend.ListAggregatedFeatureSupport400JSONResponse{
				Code:    400,
				Message: err.Error(),
			}, err
		}
	} else {
	}

	page, err := s.wptMetricsStorer.ListBrowserFeatureCountMetric(
		ctx,
		string(request.Browser),
		targetMobileBrowser,
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

	resp := backend.ListAggregatedFeatureSupport200JSONResponse{
		Metadata: page.Metadata,
		Data:     page.Data,
	}
	s.operationResponseCaches.listAggregatedFeatureSupportCache.AttemptCache(ctx, request, &resp)

	return resp, nil
}

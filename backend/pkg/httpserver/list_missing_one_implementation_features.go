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

// GetDesktopsMobileProduct returns the mobile version of the given desktop browser.
func GetDesktopsMobileProduct(browser backend.BrowserPathParam) (backend.BrowserPathParam, error) {
	switch browser {
	case backend.Chrome:
		return backend.ChromeAndroid, nil
	case backend.Firefox:
		return backend.FirefoxAndroid, nil
	case backend.Safari:
		return backend.SafariIos, nil
	case backend.Edge, backend.ChromeAndroid, backend.FirefoxAndroid, backend.SafariIos:
		return backend.BrowserPathParam(""), ErrNoMatchingMobileBrowser
	}

	return backend.BrowserPathParam(""), ErrNoMatchingMobileBrowser
}

// ListMissingOneImplementationFeatures implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListMissingOneImplementationFeatures(
	ctx context.Context,
	request backend.ListMissingOneImplementationFeaturesRequestObject) (
	backend.ListMissingOneImplementationFeaturesResponseObject, error) {

	var otherBrowsers []string
	var targetMobileBrowser *string
	if request.Params.IncludeBaselineMobileBrowsers != nil {
		otherBrowsers = make([]string, len(request.Params.Browser)*2)
		var err error
		matchingMobileBrowser, err := getDesktopsMobileProduct(request.Browser)
		if err != nil {
			return backend.ListMissingOneImplementationFeatures400JSONResponse{
				Code:    400,
				Message: err.Error(),
			}, nil
		}
		targetMobileBrowser = (*string)(&matchingMobileBrowser)

		var matchingMobileOtherBrowser backend.BrowserPathParam
		for i := range request.Params.Browser {
			otherBrowsers[i*2] = string(request.Params.Browser[i])
			matchingMobileOtherBrowser, err = getDesktopsMobileProduct(request.Params.Browser[i])
			if err != nil {
				return backend.ListMissingOneImplementationFeatures400JSONResponse{
					Code:    400,
					Message: err.Error(),
				}, nil
			}
			otherBrowsers[i*2+1] = string(matchingMobileOtherBrowser)
		}
	} else {
		otherBrowsers = make([]string, len(request.Params.Browser))
		for i := range request.Params.Browser {
			otherBrowsers[i] = string(request.Params.Browser[i])
		}
	}

	page, err := s.wptMetricsStorer.ListMissingOneImplementationFeatures(
		ctx,
		string(request.Browser),
		targetMobileBrowser,
		otherBrowsers,
		request.Date.Time,
		getPageSizeOrDefault(request.Params.PageSize),
		request.Params.PageToken,
	)
	if err != nil {
		if errors.Is(err, backendtypes.ErrInvalidPageToken) {
			slog.WarnContext(ctx, "invalid page token", "token", request.Params.PageToken, "error", err)

			return backend.ListMissingOneImplementationFeatures400JSONResponse{
				Code:    400,
				Message: "invalid page token",
			}, nil
		}

		slog.ErrorContext(ctx, "unable to get missing one implementation feature list", "error", err)

		return backend.ListMissingOneImplementationFeatures500JSONResponse{
			Code:    500,
			Message: "unable to get missing one implementation feature list",
		}, nil
	}

	resp := backend.ListMissingOneImplementationFeatures200JSONResponse{
		Metadata: page.Metadata,
		Data:     page.Data,
	}

	return resp, nil
}

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

type MissingOneBrowserParams struct {
	targetBrowser       string
	targetMobileBrowser *string
	otherBrowsers       []string
}

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

// PrepareMissingOneBrowserParams takes the raw request arguments for "missing in one browser" requests
// and formats them.
func PrepareMissingOneBrowserParams(
	targetBrowserParam backend.BrowserPathParam,
	otherBrowsersParam []backend.BrowserPathParam,
	includeMobileBrowsers bool,
) (*MissingOneBrowserParams, error) {
	var otherBrowsers []string
	var targetMobileBrowser *string
	if includeMobileBrowsers {
		var err error
		matchingMobileBrowser, err := getDesktopsMobileProduct(targetBrowserParam)
		if err != nil {
			return nil, err
		}
		targetMobileBrowser = (*string)(&matchingMobileBrowser)

		// Other browsers will include their mobile equivalents, so we'll need twice the size.
		otherBrowsers = make([]string, len(otherBrowsersParam)*2)
		var matchingMobileOtherBrowser backend.BrowserPathParam
		for i := range otherBrowsersParam {
			otherBrowsers[i*2] = string(otherBrowsersParam[i])
			matchingMobileOtherBrowser, err = getDesktopsMobileProduct(otherBrowsersParam[i])
			if err != nil {
				return nil, err
			}
			otherBrowsers[i*2+1] = string(matchingMobileOtherBrowser)
		}
	} else {
		otherBrowsers = make([]string, len(otherBrowsersParam))
		for i := range otherBrowsersParam {
			otherBrowsers[i] = string(otherBrowsersParam[i])
		}
	}

	return &MissingOneBrowserParams{
		targetBrowser:       string(targetBrowserParam),
		targetMobileBrowser: targetMobileBrowser,
		otherBrowsers:       otherBrowsers,
	}, nil
}

// ListMissingOneImplementationFeatures implements backend.StrictServerInterface.
// nolint: ireturn // Signature generated from openapi
func (s *Server) ListMissingOneImplementationFeatures(
	ctx context.Context,
	request backend.ListMissingOneImplementationFeaturesRequestObject) (
	backend.ListMissingOneImplementationFeaturesResponseObject, error) {

	browserParams, err := PrepareMissingOneBrowserParams(
		request.Browser, request.Params.Browser, request.Params.IncludeBaselineMobileBrowsers != nil)
	if err != nil {
		if errors.Is(err, ErrNoMatchingMobileBrowser) {
			return backend.ListMissingOneImplementationFeatures400JSONResponse{
				Code:    400,
				Message: err.Error(),
			}, nil
		}

		return backend.ListMissingOneImplementationFeatures500JSONResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	page, err := s.wptMetricsStorer.ListMissingOneImplementationFeatures(
		ctx,
		browserParams.targetBrowser,
		browserParams.targetMobileBrowser,
		browserParams.otherBrowsers,
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

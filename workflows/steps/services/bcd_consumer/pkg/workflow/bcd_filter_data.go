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

package workflow

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner/spanneradapters/bcdconsumertypes"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/bcd_consumer/pkg/data"
)

// ErrMissingBrowser indicates the filtered browser was not found in the bcd data.
var ErrMissingBrowser = errors.New("browser not found in bcd data")

// ErrMalformedReleaseDate indicates that the release date could not be parsed.
var ErrMalformedReleaseDate = errors.New("release date is in unexpected format")

// ErrUnknownBrowserFilter indicates that the filter is unknown.
// Developers may need to update lib/gcpspanner/spanneradapters/bcdconsumertypes/types.go.
var ErrUnknownBrowserFilter = errors.New("specified browser filter is unknown")

// ErrNoBrowserFiltersPresent indicates that no filters are present.
var ErrNoBrowserFiltersPresent = errors.New("no browser filters present")

type BCDDataFilter struct{}

func (f BCDDataFilter) checkBrowserFilters(filteredBrowsers []string) error {
	if len(filteredBrowsers) == 0 {
		return ErrNoBrowserFiltersPresent
	}

	for _, browser := range filteredBrowsers {
		browserName := bcdconsumertypes.BrowserName(browser)
		// exhaustive linter in golangci-lint will catch missing enums as they are added to
		// lib/gcpspanner/spanneradapters/bcdconsumertypes/types.go.
		switch browserName {
		case bcdconsumertypes.Chrome,
			bcdconsumertypes.Edge,
			bcdconsumertypes.Firefox,
			bcdconsumertypes.Safari,
			bcdconsumertypes.ChromeAndroid,
			bcdconsumertypes.FirefoxAndroid,
			bcdconsumertypes.SafariIos:
			continue
		default:
			return errors.Join(ErrUnknownBrowserFilter)
		}
	}

	return nil
}

// TODO: Pass in context to be used by slog.ErrorContext.
func (f BCDDataFilter) FilterData(
	in *data.BCDData, filteredBrowsers []string) ([]bcdconsumertypes.BrowserRelease, error) {
	err := f.checkBrowserFilters(filteredBrowsers)
	if err != nil {
		return nil, err
	}

	if in == nil {
		return nil, nil
	}
	var ret []bcdconsumertypes.BrowserRelease
	for _, browser := range filteredBrowsers {
		browserData, found := in.Browsers[browser]
		if !found {
			return nil, errors.Join(ErrMissingBrowser, fmt.Errorf("unable to find browser %s", browser))
		}

		for release, releaseData := range browserData.Releases {
			if releaseData.ReleaseDate == nil {
				// Maybe this might happen if the browser release is anticipated but not released yet.
				slog.Warn("data is incomplete. missing release date", "browser", browser, "release", release)

				continue
			}
			releaseDate, err := time.Parse(time.DateOnly, *releaseData.ReleaseDate)
			if err != nil {
				slog.Error("unable to parse date", "browser", browser, "release", release, "date", *releaseData.ReleaseDate)

				return nil, ErrMalformedReleaseDate
			}
			ret = append(ret, bcdconsumertypes.BrowserRelease{
				BrowserName:    bcdconsumertypes.BrowserName(browser),
				BrowserVersion: release,
				ReleaseDate:    releaseDate,
			})
		}
	}

	return ret, nil
}

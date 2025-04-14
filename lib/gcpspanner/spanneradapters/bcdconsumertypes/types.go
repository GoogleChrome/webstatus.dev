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

package bcdconsumertypes

import (
	"errors"
	"time"
)

// BrowserRelease is the representation of the metric that comes from the BCD Consumer
// This is located in the shared lib package so that it can be used in the adapter and the workflow.
type BrowserRelease struct {
	BrowserName    BrowserName
	BrowserVersion string
	ReleaseDate    time.Time
}

// BrowserName is an enumeration of the high-level keys found in the data.json
// for the browsers. The json schema itself does not list these names explicitly
// so we maintain our own subset here.
type BrowserName string

const (
	Chrome         BrowserName = "chrome"
	Edge           BrowserName = "edge"
	Firefox        BrowserName = "firefox"
	Safari         BrowserName = "safari"
	ChromeAndroid  BrowserName = "chrome_android"
	FirefoxAndroid BrowserName = "firefox_android"
	SafariIos      BrowserName = "safari_ios"
)

// ErrUnableToStoreBrowserRelease indicates that the storage layer was unable to save
// the browser release.
var ErrUnableToStoreBrowserRelease = errors.New("unable to store browser release")

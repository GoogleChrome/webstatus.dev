// Copyright 2025 Google LLC
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

// Named comparables instead of comparable to not conflict with the standard library's "comparable" interface
package comparables

import (
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/generic"
)

type Feature struct {
	ID             string
	Name           generic.OptionallySet[string]
	BaselineStatus generic.OptionallySet[BaselineState]
	BrowserImpls   generic.OptionallySet[BrowserImplementations]
	Docs           generic.OptionallySet[Docs]
}

type BaselineState struct {
	Status   generic.OptionallySet[backend.BaselineInfoStatus]
	LowDate  generic.OptionallySet[*time.Time]
	HighDate generic.OptionallySet[*time.Time]
}

// BrowserImplementations defines the specific browsers we track.
type BrowserImplementations struct {
	Chrome         generic.OptionallySet[BrowserState]
	ChromeAndroid  generic.OptionallySet[BrowserState]
	Edge           generic.OptionallySet[BrowserState]
	Firefox        generic.OptionallySet[BrowserState]
	FirefoxAndroid generic.OptionallySet[BrowserState]
	Safari         generic.OptionallySet[BrowserState]
	SafariIos      generic.OptionallySet[BrowserState]
}

// BrowserState captures the implementation details for a specific browser.
type BrowserState struct {
	Status  generic.OptionallySet[backend.BrowserImplementationStatus]
	Date    generic.OptionallySet[*time.Time]
	Version generic.OptionallySet[*string]
}

type MdnDoc struct {
	URL   generic.OptionallySet[string]
	Title generic.OptionallySet[*string]
	Slug  generic.OptionallySet[*string]
}

type Docs struct {
	MdnDocs generic.OptionallySet[[]MdnDoc]
}

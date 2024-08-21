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

package spanneradapters

import (
	"context"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
)

// ChromiumHistogramEnumConsumer handles the conversion of histogram between the workflow/API input
// format and the format used by the GCP Spanner client.
type ChromiumHistogramEnumConsumer struct {
	client WebFeatureSpannerClient
}

// NewChromiumHistogramEnumConsumer constructs an adapter for the chromium histogram enum consumer service.
func NewChromiumHistogramEnumConsumer(client WebFeatureSpannerClient) *ChromiumHistogramEnumConsumer {
	return &ChromiumHistogramEnumConsumer{client: client}
}

// ChromiumHistogramEnumsClient expects a subset of the functionality from lib/gcpspanner that only apply to
// Chromium Histograms.
type ChromiumHistogramEnumsClient interface {
	UpsertChromiumHistogramEnum(context.Context, gcpspanner.ChromiumHistogramEnum) error
}

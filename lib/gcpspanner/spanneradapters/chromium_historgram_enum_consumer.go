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
	"log/slog"
	"strings"
	"unicode"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

// ChromiumHistogramEnumConsumer handles the conversion of histogram between the workflow/API input
// format and the format used by the GCP Spanner client.
type ChromiumHistogramEnumConsumer struct {
	client ChromiumHistogramEnumsClient
}

// NewChromiumHistogramEnumConsumer constructs an adapter for the chromium histogram enum consumer service.
func NewChromiumHistogramEnumConsumer(client ChromiumHistogramEnumsClient) *ChromiumHistogramEnumConsumer {
	return &ChromiumHistogramEnumConsumer{client: client}
}

// ChromiumHistogramEnumsClient expects a subset of the functionality from lib/gcpspanner that only apply to
// Chromium Histograms.
type ChromiumHistogramEnumsClient interface {
	UpsertChromiumHistogramEnum(context.Context, gcpspanner.ChromiumHistogramEnum) (*string, error)
	UpsertChromiumHistogramEnumValue(context.Context, gcpspanner.ChromiumHistogramEnumValue) (*string, error)
	UpsertWebFeatureChromiumHistogramEnumValue(context.Context, gcpspanner.WebFeatureChromiumHistogramEnumValue) error
	GetIDFromFeatureKey(context.Context, *gcpspanner.FeatureIDFilter) (*string, error)
}

func (c *ChromiumHistogramEnumConsumer) SaveHistogramEnums(
	ctx context.Context, data metricdatatypes.HistogramMapping) error {
	for histogram, enums := range data {
		enumID, err := c.client.UpsertChromiumHistogramEnum(ctx, gcpspanner.ChromiumHistogramEnum{
			HistogramName: string(histogram),
		})
		if err != nil {
			return err
		}
		for _, enum := range enums {
			enumValueID, err := c.client.UpsertChromiumHistogramEnumValue(ctx, gcpspanner.ChromiumHistogramEnumValue{
				ChromiumHistogramEnumID: *enumID,
				BucketID:                enum.BucketID,
				Label:                   enum.Label,
			})
			if err != nil {
				return err
			}
			featureKey := enumLabelToFeatureKey(enum.Label)
			featureID, err := c.client.GetIDFromFeatureKey(
				ctx, gcpspanner.NewFeatureKeyFilter(featureKey))
			if err != nil {
				slog.WarnContext(ctx,
					"unable to find feature ID. skipping mapping",
					"error", err,
					"featureKey", featureKey,
					"label", enum.Label)

				continue
			}
			err = c.client.UpsertWebFeatureChromiumHistogramEnumValue(ctx, gcpspanner.WebFeatureChromiumHistogramEnumValue{
				WebFeatureID:                 *featureID,
				ChromiumHistogramEnumValueID: *enumValueID,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func enumLabelToFeatureKey(label string) string {
	b := strings.Builder{}
	for idx, c := range label {
		// First character just return the lower case version of it.
		if idx == 0 {
			b.WriteRune(unicode.ToLower(c))

			continue
		}
		if unicode.IsUpper(c) {
			b.WriteRune('-')
			b.WriteRune(unicode.ToLower(c))

			continue
		}
		b.WriteRune(c)
	}

	return b.String()
}

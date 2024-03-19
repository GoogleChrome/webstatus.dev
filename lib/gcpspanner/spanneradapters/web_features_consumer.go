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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

type WebFeatureSpannerClient interface {
	UpsertWebFeature(ctx context.Context, feature gcpspanner.WebFeature) error
	UpsertFeatureBaselineStatus(ctx context.Context, status gcpspanner.FeatureBaselineStatus) error
}

// NewWebFeaturesConsumer constructs an adapter for the web features consumer service.
func NewWebFeaturesConsumer(client WebFeatureSpannerClient) *WebFeaturesConsumer {
	return &WebFeaturesConsumer{client: client}
}

type WebFeaturesConsumer struct {
	client WebFeatureSpannerClient
}

func (c *WebFeaturesConsumer) InsertWebFeatures(
	ctx context.Context,
	data map[string]web_platform_dx__web_features.FeatureData) error {
	for featureID, featureData := range data {
		webFeature := gcpspanner.WebFeature{
			FeatureID: featureID,
			Name:      featureData.Name,
		}

		err := c.client.UpsertWebFeature(ctx, webFeature)
		if err != nil {
			return err
		}

		featureBaselineStatus := gcpspanner.FeatureBaselineStatus{
			FeatureID: featureID,
			Status:    getBaselineStatusEnum(featureData.Status),
			LowDate:   nil,
			HighDate:  nil,
		}
		if featureData.Status != nil {
			featureBaselineStatus.LowDate = convertStringToDate(featureData.Status.BaselineLowDate)
			// Does not work because the defs.schema.json does not have a baseline high date.
			// featureBaselineStatus.HighDate = convertStringToDate(featureData.Status.BaselineHighDate)
		}

		err = c.client.UpsertFeatureBaselineStatus(ctx, featureBaselineStatus)
		if err != nil {
			return err
		}
	}

	return nil
}

func convertStringToDate(in *string) *time.Time {
	if in == nil {
		return nil
	}

	t, err := time.Parse(time.DateOnly, *in)
	if err != nil {
		slog.Warn("unable to parse time", "time", *in)

		return nil
	}

	return &t
}

func getBaselineStatusEnum(status *web_platform_dx__web_features.Status) gcpspanner.BaselineStatus {
	if status == nil || status.Baseline == nil {
		return gcpspanner.BaselineStatusUndefined
	}
	if status.Baseline.Enum != nil {
		switch *status.Baseline.Enum {
		case web_platform_dx__web_features.High:
			return gcpspanner.BaselineStatusHigh
		case web_platform_dx__web_features.Low:
			return gcpspanner.BaselineStatusLow
		}
	} else if status.Baseline.Bool != nil && !*status.Baseline.Bool {
		return gcpspanner.BaselineStatusNone
	}

	return gcpspanner.BaselineStatusUndefined
}

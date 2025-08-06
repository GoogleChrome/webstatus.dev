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

// WebFeatureSpannerClient expects a subset of the functionality from lib/gcpspanner that only apply to WebFeatures.
type WebFeatureSpannerClient interface {
	SyncWebFeatures(ctx context.Context, features []gcpspanner.WebFeature) error
	FetchAllWebFeatureIDsAndKeys(ctx context.Context) ([]gcpspanner.SpannerFeatureIDAndKey, error)
	UpsertFeatureBaselineStatus(ctx context.Context, featureID string, status gcpspanner.FeatureBaselineStatus) error
	UpsertBrowserFeatureAvailability(
		ctx context.Context,
		featureID string,
		featureAvailability gcpspanner.BrowserFeatureAvailability) error
	UpsertFeatureSpec(ctx context.Context, webFeatureID string, input gcpspanner.FeatureSpec) error
	UpsertFeatureDiscouragedDetails(ctx context.Context, featureID string,
		in gcpspanner.FeatureDiscouragedDetails) error
	PrecalculateBrowserFeatureSupportEvents(ctx context.Context, startAt, endAt time.Time) error
}

// NewWebFeaturesConsumer constructs an adapter for the web features consumer service.
func NewWebFeaturesConsumer(client WebFeatureSpannerClient) *WebFeaturesConsumer {
	return &WebFeaturesConsumer{client: client}
}

// WebFeaturesConsumer handles the conversion of web feature data between the workflow/API input
// format and the format used by the GCP Spanner client.
type WebFeaturesConsumer struct {
	client WebFeatureSpannerClient
}

func (c *WebFeaturesConsumer) InsertWebFeatures(
	ctx context.Context,
	data map[string]web_platform_dx__web_features.FeatureValue,
	startAt, endAt time.Time) (map[string]string, error) {
	// 1. Prepare all WebFeature structs from the input data.
	allFeatures := make([]gcpspanner.WebFeature, 0, len(data))
	for featureID, featureData := range data {
		webFeature := gcpspanner.WebFeature{
			FeatureKey:      featureID,
			Name:            featureData.Name,
			Description:     featureData.Description,
			DescriptionHTML: featureData.DescriptionHTML,
		}
		allFeatures = append(allFeatures, webFeature)
	}

	// 2. Sync all features at once. This will insert, update, and delete features
	// to make the database match the desired state.
	if err := c.client.SyncWebFeatures(ctx, allFeatures); err != nil {
		slog.ErrorContext(ctx, "failed to sync web features", "error", err)

		return nil, err
	}

	// 3. Loop through the data again to process all related entities for each feature.
	for featureID, featureData := range data {
		featureBaselineStatus := gcpspanner.FeatureBaselineStatus{
			Status:   getBaselineStatusEnum(featureData.Status),
			LowDate:  nil,
			HighDate: nil,
		}

		featureBaselineStatus.LowDate = convertStringToDate(featureData.Status.BaselineLowDate)
		featureBaselineStatus.HighDate = convertStringToDate(featureData.Status.BaselineHighDate)

		err := c.client.UpsertFeatureBaselineStatus(ctx, featureID, featureBaselineStatus)
		if err != nil {
			return nil, err
		}

		// Read the browser support data.
		fba := extractBrowserAvailability(featureData)
		for _, browserAvailability := range fba {
			err := c.client.UpsertBrowserFeatureAvailability(ctx, featureID, browserAvailability)
			if err != nil {
				slog.ErrorContext(ctx, "unable to insert BrowserFeatureAvailability",
					"browserName", browserAvailability.BrowserName,
					"browserVersion", browserAvailability.BrowserVersion,
					"featureID", featureID,
				)

				return nil, err
			}
		}

		// Read the spec information
		err = consumeFeatureSpecInformation(ctx, c.client, featureID, featureData)
		if err != nil {
			return nil, err
		}

		if featureData.Discouraged != nil {
			err = c.client.UpsertFeatureDiscouragedDetails(ctx, featureID, gcpspanner.FeatureDiscouragedDetails{
				AccordingTo:  featureData.Discouraged.AccordingTo,
				Alternatives: featureData.Discouraged.Alternatives,
			})
			if err != nil {
				slog.ErrorContext(ctx, "unable to insert Discouraged Details",
					"discoruagedDetails", featureData.Discouraged,
					"featureID", featureID,
				)

				return nil, err
			}
		}
	}

	// 4. Now that all the feature information is stored, run pre-calculation of
	// feature support events.
	err := c.client.PrecalculateBrowserFeatureSupportEvents(ctx, startAt, endAt)
	if err != nil {
		return nil, err
	}

	// 5. Fetch all feature IDs to construct the return map.
	idAndKeyPairs, err := c.client.FetchAllWebFeatureIDsAndKeys(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch feature IDs and keys after sync", "error", err)

		return nil, err
	}
	ret := make(map[string]string, len(idAndKeyPairs))
	for _, pair := range idAndKeyPairs {
		ret[pair.FeatureKey] = pair.ID
	}

	return ret, nil
}

func consumeFeatureSpecInformation(ctx context.Context,
	client WebFeatureSpannerClient,
	featureID string,
	featureData web_platform_dx__web_features.FeatureValue) error {
	if featureData.Spec == nil {
		return nil
	}

	var links []string
	if featureData.Spec.String != nil {
		links = []string{*featureData.Spec.String}
	} else if len(featureData.Spec.StringArray) > 0 {
		links = featureData.Spec.StringArray
	}

	if len(links) > 0 {
		spec := gcpspanner.FeatureSpec{
			Links: links,
		}
		err := client.UpsertFeatureSpec(ctx, featureID, spec)
		if err != nil {
			slog.ErrorContext(ctx,
				"unable to insert FeatureSpec",
				"links", spec.Links,
				"featureID", featureID,
				"error", err,
			)

			return err
		}
	}

	return nil

}

func extractBrowserAvailability(
	featureData web_platform_dx__web_features.FeatureValue) []gcpspanner.BrowserFeatureAvailability {
	var fba []gcpspanner.BrowserFeatureAvailability
	support := featureData.Status.Support
	if support.Chrome != nil {
		fba = append(fba, gcpspanner.BrowserFeatureAvailability{
			BrowserName:    "chrome",
			BrowserVersion: *support.Chrome,
		})
	}
	if support.Edge != nil {
		fba = append(fba, gcpspanner.BrowserFeatureAvailability{
			BrowserName:    "edge",
			BrowserVersion: *support.Edge,
		})
	}
	if support.Firefox != nil {
		fba = append(fba, gcpspanner.BrowserFeatureAvailability{
			BrowserName:    "firefox",
			BrowserVersion: *support.Firefox,
		})
	}
	if support.Safari != nil {
		fba = append(fba, gcpspanner.BrowserFeatureAvailability{
			BrowserName:    "safari",
			BrowserVersion: *support.Safari,
		})
	}
	if support.ChromeAndroid != nil {
		fba = append(fba, gcpspanner.BrowserFeatureAvailability{
			BrowserName:    "chrome_android",
			BrowserVersion: *support.ChromeAndroid,
		})
	}
	if support.FirefoxAndroid != nil {
		fba = append(fba, gcpspanner.BrowserFeatureAvailability{
			BrowserName:    "firefox_android",
			BrowserVersion: *support.FirefoxAndroid,
		})
	}
	if support.SafariIos != nil {
		fba = append(fba, gcpspanner.BrowserFeatureAvailability{
			BrowserName:    "safari_ios",
			BrowserVersion: *support.SafariIos,
		})
	}

	return fba
}

// convertStringToDate converts a date string (in DateOnly format) to a time.Time pointer.
// Handles potential parsing errors and returns nil if the input string is nil.
func convertStringToDate(in *string) *time.Time {
	if in == nil {
		return nil
	}

	t, err := time.Parse(time.DateOnly, *in)
	if err != nil {
		slog.WarnContext(context.TODO(), "unable to parse time", "time", *in)

		return nil
	}

	return &t
}

// getBaselineStatusEnum converts the web feature status to the Spanner-compatible BaselineStatus type.
func getBaselineStatusEnum(status web_platform_dx__web_features.Status) *gcpspanner.BaselineStatus {
	if status.Baseline == nil {
		return nil
	}
	if status.Baseline.Enum != nil {
		switch *status.Baseline.Enum {
		case web_platform_dx__web_features.High:
			return valuePtr(gcpspanner.BaselineStatusHigh)
		case web_platform_dx__web_features.Low:
			return valuePtr(gcpspanner.BaselineStatusLow)
		}
	} else if status.Baseline.Bool != nil && !*status.Baseline.Bool {
		return valuePtr(gcpspanner.BaselineStatusNone)
	}

	return nil
}

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
	"errors"
	"log/slog"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	FetchAllFeatureKeys(context.Context) ([]string, error)
	GetAllMovedWebFeatures(ctx context.Context) ([]gcpspanner.MovedWebFeature, error)
}

// Used by GCP Log-based metrics to extract the data about mismatch mappings.
const logMissingFeatureIDMetricMsg = "unable to find feature ID. skipping mapping"

func (c *ChromiumHistogramEnumConsumer) GetAllMovedWebFeatures(
	ctx context.Context) (map[string]webdxfeaturetypes.FeatureMovedData, error) {
	movedFeatures, err := c.client.GetAllMovedWebFeatures(ctx)
	if err != nil {
		return nil, err
	}

	return convertGCPSpannerMovedFeaturesToMap(movedFeatures), nil
}

func (c *ChromiumHistogramEnumConsumer) SaveHistogramEnums(
	ctx context.Context, data metricdatatypes.HistogramMapping) error {
	featureKeys, err := c.client.FetchAllFeatureKeys(ctx)
	if err != nil {
		return errors.Join(ErrFailedToGetFeatureKeys, err)
	}
	enumToFeatureKeyMap := createEnumToFeatureKeyMap(featureKeys)

	histogramsToEnumMap, histogramsToAllFeatureKeySet := buildHistogramMaps(data, enumToFeatureKeyMap)

	// Get the moved features
	movedFeatures, err := c.GetAllMovedWebFeatures(ctx)
	if err != nil {
		return err
	}

	err = migrateMovedFeaturesForChromiumHistograms(
		ctx, histogramsToEnumMap, histogramsToAllFeatureKeySet, movedFeatures)
	if err != nil {
		return err
	}

	// Create mapping of anticipated enums to feature keys
	for histogram, enums := range data {
		enumID, err := c.client.UpsertChromiumHistogramEnum(ctx, gcpspanner.ChromiumHistogramEnum{
			HistogramName: string(histogram),
		})
		if err != nil {
			return errors.Join(ErrFailedToStoreEnum, err)
		}
		for _, enum := range enums {
			enumValueID, err := c.client.UpsertChromiumHistogramEnumValue(ctx, gcpspanner.ChromiumHistogramEnumValue{
				ChromiumHistogramEnumID: *enumID,
				BucketID:                enum.Value,
				Label:                   enum.Label,
			})
			if err != nil {
				return errors.Join(ErrFailedToStoreEnumValue, err)
			}

			featureKey := histogramsToEnumMap[histogram][enum.Value]
			if featureKey == nil {
				continue
			}

			featureID, err := c.client.GetIDFromFeatureKey(
				ctx, gcpspanner.NewFeatureKeyFilter(*featureKey))
			if err != nil {
				slog.WarnContext(ctx,
					logMissingFeatureIDMetricMsg,
					"error", err,
					"featureKey", *featureKey,
					"label", enum.Label)

				continue
			}
			err = c.client.UpsertWebFeatureChromiumHistogramEnumValue(ctx,
				gcpspanner.WebFeatureChromiumHistogramEnumValue{
					WebFeatureID:                 *featureID,
					ChromiumHistogramEnumValueID: *enumValueID,
				})
			if err != nil {
				return errors.Join(ErrFailedToStoreEnumValueWebFeatureMapping, err)
			}
		}
	}

	return nil
}

func buildHistogramMaps(
	data metricdatatypes.HistogramMapping,
	enumToFeatureKeyMap map[string]string,
) (map[metricdatatypes.HistogramName]map[int64]*string,
	map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue) {
	histogramsToEnumMap := map[metricdatatypes.HistogramName]map[int64]*string{}
	histogramsToAllFeatureKeySet := make(
		map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue)

	for histogram, enums := range data {
		enumIDToFeatureKeyMap := make(map[int64]*string, len(enums))
		allFeatureKeySet := make(map[string]metricdatatypes.HistogramEnumValue, len(enums))
		for _, enum := range enums {
			featureKey, found := enumToFeatureKeyMap[enum.Label]
			if !found {
				enumIDToFeatureKeyMap[enum.Value] = nil

				continue
			}
			enumIDToFeatureKeyMap[enum.Value] = &featureKey
			allFeatureKeySet[featureKey] = enum
		}
		histogramsToEnumMap[histogram] = enumIDToFeatureKeyMap
		histogramsToAllFeatureKeySet[histogram] = allFeatureKeySet
	}

	return histogramsToEnumMap, histogramsToAllFeatureKeySet
}

func migrateMovedFeaturesForChromiumHistograms(
	ctx context.Context,
	histogramsToEnumMap map[metricdatatypes.HistogramName]map[int64]*string,
	histogramsToAllFeatureKeySet map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue,
	movedFeatures map[string]webdxfeaturetypes.FeatureMovedData,
) error {
	for histogram, allFeaturesKeySet := range histogramsToAllFeatureKeySet {
		logger := slog.With("histogram", histogram)

		err := NewMigrator(
			movedFeatures,
			allFeaturesKeySet,
			histogramsToEnumMap[histogram],
			WithLoggerForMigrator[metricdatatypes.HistogramEnumValue, map[int64]*string](logger),
		).Migrate(ctx, func(oldKey, newKey string, data map[int64]*string) {
			data[allFeaturesKeySet[oldKey].Value] = &newKey
		})
		if err != nil {
			return err
		}
	}

	return nil
}

var (
	// ErrFailedToStoreEnum indicates the storage layer failed to store chromium enum.
	ErrFailedToStoreEnum = errors.New("failed to store chromium enum")
	// ErrFailedToStoreEnumValue indicates the storage layer failed to store chromium enum value.
	ErrFailedToStoreEnumValue = errors.New("failed to store chromium enum value")
	// ErrFailedToStoreEnumValueWebFeatureMapping indicates the storage layer failed to store
	// the mapping between enum value and web feature.
	ErrFailedToStoreEnumValueWebFeatureMapping = errors.New(
		"failed to store web feature to chromium enum value mapping")
	// ErrFailedToGetFeatureKeys indicates an internal error when trying to get all the feature keys.
	ErrFailedToGetFeatureKeys = errors.New("failed to get feature keys")
)

// nolint:lll // WONTFIX: useful comment message
// createEnumToFeatureKeyMap uses the list of WebDX feature keys to
// generate a map from the enum label (e.g., "ViewTransitions")
// back to its original WebDX feature key (e.g., "view-transitions").
// It uses the same transformation logic described in the Chromium mojom file.
// https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/mojom/use_counter/metrics/webdx_feature.mojom;l=35-47;drc=822a70f9ac61a75babe9d24ddfc32ab475acc7e1
func createEnumToFeatureKeyMap(featureKeys []string) map[string]string {
	titleCaser := cases.Title(language.English)
	m := make(map[string]string, len(featureKeys))
	specialCases := map[string]string{
		"float16array":          "Float16Array",
		"uint8array-base64-hex": "Uint8ArrayBase64Hex",
	}
	for _, featureKey := range featureKeys {
		if specialCaseLabel, found := specialCases[featureKey]; found {
			m[specialCaseLabel] = featureKey

			continue
		}

		enumLabel := titleCaser.String(featureKey)
		enumLabel = strings.ReplaceAll(enumLabel, "-", "")
		// Before storing it, check if it exists
		m[enumLabel] = featureKey
	}

	return m
}

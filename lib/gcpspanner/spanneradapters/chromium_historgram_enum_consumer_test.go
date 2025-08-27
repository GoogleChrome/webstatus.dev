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
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

type mockChromiumHistogramEnumsClient struct {
	upsertChromiumHistogramEnum func(context.Context,
		gcpspanner.ChromiumHistogramEnum) (*string, error)
	upsertChromiumHistogramEnumValue func(context.Context,
		gcpspanner.ChromiumHistogramEnumValue) (*string, error)
	upsertWebFeatureChromiumHistogramEnumValue func(context.Context,
		gcpspanner.WebFeatureChromiumHistogramEnumValue) error
	getIDFromFeatureKey    func(context.Context, *gcpspanner.FeatureIDFilter) (*string, error)
	fetchAllFeatureKeys    func(context.Context) ([]string, error)
	getAllMovedWebFeatures func(ctx context.Context) ([]gcpspanner.MovedWebFeature, error)
}

func (m *mockChromiumHistogramEnumsClient) UpsertChromiumHistogramEnum(ctx context.Context,
	in gcpspanner.ChromiumHistogramEnum) (*string, error) {
	return m.upsertChromiumHistogramEnum(ctx, in)
}

func (m *mockChromiumHistogramEnumsClient) UpsertChromiumHistogramEnumValue(ctx context.Context,
	in gcpspanner.ChromiumHistogramEnumValue) (*string, error) {
	return m.upsertChromiumHistogramEnumValue(ctx, in)
}

func (m *mockChromiumHistogramEnumsClient) UpsertWebFeatureChromiumHistogramEnumValue(ctx context.Context,
	in gcpspanner.WebFeatureChromiumHistogramEnumValue) error {
	return m.upsertWebFeatureChromiumHistogramEnumValue(ctx, in)
}

func (m *mockChromiumHistogramEnumsClient) GetIDFromFeatureKey(ctx context.Context,
	in *gcpspanner.FeatureIDFilter) (*string, error) {
	return m.getIDFromFeatureKey(ctx, in)
}

func (m *mockChromiumHistogramEnumsClient) FetchAllFeatureKeys(
	ctx context.Context) ([]string, error) {
	return m.fetchAllFeatureKeys(ctx)
}

func (m *mockChromiumHistogramEnumsClient) GetAllMovedWebFeatures(
	ctx context.Context) ([]gcpspanner.MovedWebFeature, error) {
	return m.getAllMovedWebFeatures(ctx)
}

func TestChromiumHistogramEnumConsumer_SaveHistogramEnums(t *testing.T) {
	tests := []struct {
		name        string
		client      *mockChromiumHistogramEnumsClient
		data        metricdatatypes.HistogramMapping
		expectedErr error
	}{
		{
			name: "Success",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return valuePtr("enumID"), nil
				},
				upsertChromiumHistogramEnumValue: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnumValue) (*string, error) {
					return valuePtr("enumValueID"), nil
				},
				getIDFromFeatureKey: func(_ context.Context,
					_ *gcpspanner.FeatureIDFilter) (*string, error) {
					return valuePtr("featureID"), nil
				},
				upsertWebFeatureChromiumHistogramEnumValue: func(_ context.Context,
					_ gcpspanner.WebFeatureChromiumHistogramEnumValue) error {
					return nil
				},
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return nil, nil
				},
			},
			data: metricdatatypes.HistogramMapping{
				metricdatatypes.WebDXFeatureEnum: []metricdatatypes.HistogramEnumValue{
					{Value: 1, Label: "EnumLabel"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "FetchAllFeatureKeys returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return nil, errors.New("test error")
				},
				upsertChromiumHistogramEnum:                nil,
				upsertChromiumHistogramEnumValue:           nil,
				upsertWebFeatureChromiumHistogramEnumValue: nil,
				getIDFromFeatureKey:                        nil,
				getAllMovedWebFeatures:                     nil,
			},
			data: metricdatatypes.HistogramMapping{
				metricdatatypes.WebDXFeatureEnum: []metricdatatypes.HistogramEnumValue{
					{Value: 1, Label: "EnumLabel"},
				},
			},
			expectedErr: ErrFailedToGetFeatureKeys,
		},
		{
			name: "UpsertChromiumHistogramEnum returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return nil, errors.New("test error")
				},
				upsertChromiumHistogramEnumValue:           nil,
				upsertWebFeatureChromiumHistogramEnumValue: nil,
				getIDFromFeatureKey:                        nil,
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return nil, nil
				},
			},
			data: metricdatatypes.HistogramMapping{
				metricdatatypes.WebDXFeatureEnum: []metricdatatypes.HistogramEnumValue{
					{Value: 1, Label: "EnumLabel"},
				},
			},
			expectedErr: ErrFailedToStoreEnum,
		},
		{
			name: "UpsertChromiumHistogramEnumValue returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return valuePtr("enumID"), nil
				},
				upsertChromiumHistogramEnumValue: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnumValue) (*string, error) {
					return nil, errors.New("test error")
				},
				upsertWebFeatureChromiumHistogramEnumValue: nil,
				getIDFromFeatureKey:                        nil,
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return nil, nil
				},
			},
			data: metricdatatypes.HistogramMapping{
				metricdatatypes.WebDXFeatureEnum: []metricdatatypes.HistogramEnumValue{
					{Value: 1, Label: "EnumLabel"},
				},
			},
			expectedErr: ErrFailedToStoreEnumValue,
		},
		{
			name: "GetIDFromFeatureKey returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return valuePtr("enumID"), nil
				},
				upsertChromiumHistogramEnumValue: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnumValue) (*string, error) {
					return valuePtr("enumValueID"), nil
				},
				getIDFromFeatureKey: func(_ context.Context,
					_ *gcpspanner.FeatureIDFilter) (*string, error) {
					return nil, errors.New("test error")
				},
				upsertWebFeatureChromiumHistogramEnumValue: nil,
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return nil, nil
				},
			},
			data: metricdatatypes.HistogramMapping{
				metricdatatypes.WebDXFeatureEnum: []metricdatatypes.HistogramEnumValue{
					{Value: 1, Label: "EnumLabel"},
				},
			},
			expectedErr: nil,
		},
		{
			name: "UpsertWebFeatureChromiumHistogramEnumValue returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return valuePtr("enumID"), nil
				},
				upsertChromiumHistogramEnumValue: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnumValue) (*string, error) {
					return valuePtr("enumValueID"), nil
				},
				getIDFromFeatureKey: func(_ context.Context,
					_ *gcpspanner.FeatureIDFilter) (*string, error) {
					return valuePtr("featureID"), nil
				},
				upsertWebFeatureChromiumHistogramEnumValue: func(_ context.Context,
					_ gcpspanner.WebFeatureChromiumHistogramEnumValue) error {
					return errors.New("test error")
				},
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return nil, nil
				},
			},
			data: metricdatatypes.HistogramMapping{
				metricdatatypes.WebDXFeatureEnum: []metricdatatypes.HistogramEnumValue{
					{Value: 1, Label: "EnumLabel"},
				},
			},
			expectedErr: ErrFailedToStoreEnumValueWebFeatureMapping,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &ChromiumHistogramEnumConsumer{
				client: tc.client,
			}
			err := c.SaveHistogramEnums(context.Background(), tc.data)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf(
					"ChromiumHistogramEnumConsumer.SaveHistogramEnums() error = %v, expectedErr %v",
					err, tc.expectedErr)

				return
			}
		})
	}
}

func TestCreateEnumToFeatureKeyMap(t *testing.T) {
	featureKeys := []string{
		"canvas-2d-color-management",
		"http3",
		"intersection-observer-v2",
		"view-transitions",
		// Special cases
		"float16array",
		"uint8array-base64-hex",
	}
	// nolint: lll // WONTFIX: useful comment with SHA
	want := map[string]string{
		"Canvas2DColorManagement": "canvas-2d-color-management",
		"Http3":                   "http3",
		"IntersectionObserverV2":  "intersection-observer-v2",
		"ViewTransitions":         "view-transitions",
		/*
			Special cases
		*/
		// https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/mojom/use_counter/metrics/webdx_feature.mojom;l=360;drc=822a70f9ac61a75babe9d24ddfc32ab475acc7e1
		// https://github.com/web-platform-dx/web-features/blob/main/features/float16array.yml
		"Float16Array": "float16array",
		// https://source.chromium.org/chromium/chromium/src/+/main:third_party/blink/public/mojom/use_counter/metrics/webdx_feature.mojom;l=396;drc=822a70f9ac61a75babe9d24ddfc32ab475acc7e1
		// https://github.com/web-platform-dx/web-features/blob/main/features/uint8array-base64-hex.yml
		"Uint8ArrayBase64Hex": "uint8array-base64-hex",
	}
	got := createEnumToFeatureKeyMap(featureKeys)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("createEnumToFeatureKeyMap()\ngot:  (%+v)\nwant: (%+v)\n", got, want)
	}
}

func TestMigrateMovedFeatures(t *testing.T) {
	testCases := []struct {
		name                         string
		histogramsToEnumMap          map[metricdatatypes.HistogramName]map[int64]*string
		histogramsToAllFeatureKeySet map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue
		movedFeatures                map[string]web_platform_dx__web_features.FeatureMovedData
		expectedHistogramsToEnumMap  map[metricdatatypes.HistogramName]map[int64]*string
		expectedErr                  error
	}{
		{
			name: "successful migration",
			histogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{
				metricdatatypes.WebDXFeatureEnum: {
					1: valuePtr("old-feature"),
				},
			},
			histogramsToAllFeatureKeySet: map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue{
				metricdatatypes.WebDXFeatureEnum: {
					"old-feature": {Value: 1, Label: "old-feature"},
				},
			},
			movedFeatures: map[string]web_platform_dx__web_features.FeatureMovedData{
				"old-feature": {RedirectTarget: "new-feature", Kind: web_platform_dx__web_features.Moved},
			},
			expectedHistogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{
				metricdatatypes.WebDXFeatureEnum: {
					1: valuePtr("new-feature"),
				},
			},
			expectedErr: nil,
		},
		{
			name: "conflict with existing feature",
			histogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{
				metricdatatypes.WebDXFeatureEnum: {
					1: valuePtr("old-feature"),
					2: valuePtr("new-feature"),
				},
			},
			histogramsToAllFeatureKeySet: map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue{
				metricdatatypes.WebDXFeatureEnum: {
					"old-feature": {Value: 1, Label: "old-feature"},
					"new-feature": {Value: 2, Label: "new-feature"},
				},
			},
			movedFeatures: map[string]web_platform_dx__web_features.FeatureMovedData{
				"old-feature": {RedirectTarget: "new-feature", Kind: web_platform_dx__web_features.Moved},
			},
			expectedHistogramsToEnumMap: nil,
			expectedErr:                 ErrConflictMigratingFeatureKey,
		},
		{
			name: "no migration needed",
			histogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{
				metricdatatypes.WebDXFeatureEnum: {
					1: valuePtr("feature-a"),
				},
			},
			histogramsToAllFeatureKeySet: map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue{
				metricdatatypes.WebDXFeatureEnum: {
					"feature-a": {Value: 1, Label: "feature-a"},
				},
			},
			movedFeatures: map[string]web_platform_dx__web_features.FeatureMovedData{},
			expectedHistogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{
				metricdatatypes.WebDXFeatureEnum: {
					1: valuePtr("feature-a"),
				},
			},
			expectedErr: nil,
		},
		{
			name: "multiple migrations",
			histogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{
				metricdatatypes.WebDXFeatureEnum: {
					1: valuePtr("old-a"),
					2: valuePtr("feature-c"),
				},
				"hist2": {
					3: valuePtr("old-b"),
				},
			},
			histogramsToAllFeatureKeySet: map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue{
				metricdatatypes.WebDXFeatureEnum: {
					"old-a":     {Value: 1, Label: "old-a"},
					"feature-c": {Value: 2, Label: "feature-c"},
				},
				"hist2": {
					"old-b": {Value: 3, Label: "old-b"},
				},
			},
			movedFeatures: map[string]web_platform_dx__web_features.FeatureMovedData{
				"old-a": {RedirectTarget: "new-a", Kind: web_platform_dx__web_features.Moved},
				"old-b": {RedirectTarget: "new-b", Kind: web_platform_dx__web_features.Moved},
			},
			expectedHistogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{
				metricdatatypes.WebDXFeatureEnum: {
					1: valuePtr("new-a"),
					2: valuePtr("feature-c"),
				},
				"hist2": {
					3: valuePtr("new-b"),
				},
			},
			expectedErr: nil,
		},
		{
			name:                         "empty data",
			histogramsToEnumMap:          map[metricdatatypes.HistogramName]map[int64]*string{},
			histogramsToAllFeatureKeySet: map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue{},
			movedFeatures: map[string]web_platform_dx__web_features.FeatureMovedData{
				"a": {RedirectTarget: "b", Kind: web_platform_dx__web_features.Moved},
			},
			expectedHistogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{},
			expectedErr:                 nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateMovedFeatures(
				context.Background(), tc.histogramsToEnumMap, tc.histogramsToAllFeatureKeySet, tc.movedFeatures)

			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}

			if tc.expectedErr == nil && !reflect.DeepEqual(tc.histogramsToEnumMap, tc.expectedHistogramsToEnumMap) {
				t.Errorf("expected data %v, got %v", tc.expectedHistogramsToEnumMap, tc.histogramsToEnumMap)
			}
		})
	}
}

var errTestDatabaseError = errors.New("test database error")

func TestChromiumHistogramEnumConsumer_GetAllMovedWebFeatures(t *testing.T) {
	testCases := []struct {
		name          string
		mockClient    *mockChromiumHistogramEnumsClient
		expected      map[string]web_platform_dx__web_features.FeatureMovedData
		expectedError error
	}{
		{
			name: "Success",
			mockClient: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys:                        nil,
				upsertChromiumHistogramEnum:                nil,
				upsertChromiumHistogramEnumValue:           nil,
				upsertWebFeatureChromiumHistogramEnumValue: nil,
				getIDFromFeatureKey:                        nil,
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return []gcpspanner.MovedWebFeature{
						{
							OriginalFeatureKey: "feature1",
							NewFeatureKey:      "new-feature1",
						},
						{
							OriginalFeatureKey: "feature2",
							NewFeatureKey:      "new-feature2",
						},
					}, nil
				},
			},
			expected: map[string]web_platform_dx__web_features.FeatureMovedData{
				"feature1": {
					RedirectTarget: "new-feature1",
					Kind:           web_platform_dx__web_features.Moved,
				},
				"feature2": {
					RedirectTarget: "new-feature2",
					Kind:           web_platform_dx__web_features.Moved,
				},
			},
			expectedError: nil,
		},
		{
			name: "Database error",
			mockClient: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys:                        nil,
				upsertChromiumHistogramEnum:                nil,
				upsertChromiumHistogramEnumValue:           nil,
				upsertWebFeatureChromiumHistogramEnumValue: nil,
				getIDFromFeatureKey:                        nil,
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return nil, errTestDatabaseError
				},
			},
			expected:      nil,
			expectedError: errTestDatabaseError,
		},
		{
			name: "Empty result",
			mockClient: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys:                        nil,
				upsertChromiumHistogramEnum:                nil,
				upsertChromiumHistogramEnumValue:           nil,
				upsertWebFeatureChromiumHistogramEnumValue: nil,
				getIDFromFeatureKey:                        nil,
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return []gcpspanner.MovedWebFeature{}, nil
				},
			},
			expected:      map[string]web_platform_dx__web_features.FeatureMovedData{},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			consumer := &ChromiumHistogramEnumConsumer{client: tc.mockClient}
			result, err := consumer.GetAllMovedWebFeatures(context.Background())
			if !errors.Is(tc.expectedError, err) {
				t.Errorf("Unexpected error. Expected %v, got %v", tc.expectedError, err)
			}
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Unexpected result. Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

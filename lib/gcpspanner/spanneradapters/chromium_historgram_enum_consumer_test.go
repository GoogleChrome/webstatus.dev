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
	"net/http"
	"reflect"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/chromium_histogram_enums/workflow"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/web_feature_consumer/pkg/data"
)

type mockChromiumHistogramEnumsClient struct {
	upsertChromiumHistogramEnum func(context.Context,
		gcpspanner.ChromiumHistogramEnum) (*string, error)
	syncChromiumHistogramEnumValues func(context.Context,
		[]gcpspanner.ChromiumHistogramEnumValue) error
	getIDFromChromiumHistogramEnumValueKey func(
		ctx context.Context, chromiumHistogramEnumID string, bucketID int64) (*string, error)
	syncWebFeatureChromiumHistogramEnumValues func(context.Context,
		[]gcpspanner.WebFeatureChromiumHistogramEnumValue) error
	getIDFromFeatureKey    func(context.Context, *gcpspanner.FeatureIDFilter) (*string, error)
	fetchAllFeatureKeys    func(context.Context) ([]string, error)
	getAllMovedWebFeatures func(ctx context.Context) ([]gcpspanner.MovedWebFeature, error)
}

func (m *mockChromiumHistogramEnumsClient) UpsertChromiumHistogramEnum(ctx context.Context,
	in gcpspanner.ChromiumHistogramEnum) (*string, error) {
	return m.upsertChromiumHistogramEnum(ctx, in)
}

func (m *mockChromiumHistogramEnumsClient) SyncChromiumHistogramEnumValues(
	ctx context.Context, in []gcpspanner.ChromiumHistogramEnumValue) error {
	return m.syncChromiumHistogramEnumValues(ctx, in)
}

func (m *mockChromiumHistogramEnumsClient) GetIDFromChromiumHistogramEnumValueKey(
	ctx context.Context, chromiumHistogramEnumID string, bucketID int64) (*string, error) {
	return m.getIDFromChromiumHistogramEnumValueKey(ctx, chromiumHistogramEnumID, bucketID)
}

func (m *mockChromiumHistogramEnumsClient) SyncWebFeatureChromiumHistogramEnumValues(ctx context.Context,
	in []gcpspanner.WebFeatureChromiumHistogramEnumValue) error {
	return m.syncWebFeatureChromiumHistogramEnumValues(ctx, in)
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
				syncChromiumHistogramEnumValues: func(_ context.Context,
					_ []gcpspanner.ChromiumHistogramEnumValue) error {
					return nil
				},
				getIDFromChromiumHistogramEnumValueKey: func(
					_ context.Context, _ string, _ int64) (*string, error) {
					return valuePtr("enumValueID"), nil
				},
				getIDFromFeatureKey: func(_ context.Context,
					_ *gcpspanner.FeatureIDFilter) (*string, error) {
					return valuePtr("featureID"), nil
				},
				syncWebFeatureChromiumHistogramEnumValues: func(_ context.Context,
					in []gcpspanner.WebFeatureChromiumHistogramEnumValue) error {
					expected := []gcpspanner.WebFeatureChromiumHistogramEnumValue{
						{
							WebFeatureID:                 "featureID",
							ChromiumHistogramEnumValueID: "enumValueID",
						},
					}
					if !reflect.DeepEqual(in, expected) {
						t.Errorf("unexpected input to SyncWebFeatureChromiumHistogramEnumValues. got %+v, want %+v",
							in, expected)
					}

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
				upsertChromiumHistogramEnum:               nil,
				syncChromiumHistogramEnumValues:           nil,
				getIDFromChromiumHistogramEnumValueKey:    nil,
				syncWebFeatureChromiumHistogramEnumValues: nil,
				getIDFromFeatureKey:                       nil,
				getAllMovedWebFeatures:                    nil,
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
				syncChromiumHistogramEnumValues:           nil,
				getIDFromChromiumHistogramEnumValueKey:    nil,
				syncWebFeatureChromiumHistogramEnumValues: nil,
				getIDFromFeatureKey:                       nil,
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
			name: "SyncChromiumHistogramEnumValue returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return valuePtr("enumID"), nil
				},
				syncChromiumHistogramEnumValues: func(_ context.Context,
					_ []gcpspanner.ChromiumHistogramEnumValue) error {
					return errors.New("test error")
				},
				getIDFromChromiumHistogramEnumValueKey:    nil,
				syncWebFeatureChromiumHistogramEnumValues: nil,
				getIDFromFeatureKey:                       nil,
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
			name: "GetIDFromChromiumHistogramEnumValueKey returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return valuePtr("enumID"), nil
				},
				syncChromiumHistogramEnumValues: func(_ context.Context,
					_ []gcpspanner.ChromiumHistogramEnumValue) error {
					return nil
				},
				getIDFromChromiumHistogramEnumValueKey: func(
					_ context.Context, _ string, _ int64) (*string, error) {
					return nil, errors.New("test error")
				},
				syncWebFeatureChromiumHistogramEnumValues: nil,
				getIDFromFeatureKey:                       nil,
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
				syncChromiumHistogramEnumValues: func(_ context.Context,
					_ []gcpspanner.ChromiumHistogramEnumValue) error {
					return nil
				},
				getIDFromChromiumHistogramEnumValueKey: func(
					_ context.Context, _ string, _ int64) (*string, error) {
					return valuePtr("enumValueID"), nil
				},
				getIDFromFeatureKey: func(_ context.Context,
					_ *gcpspanner.FeatureIDFilter) (*string, error) {
					return nil, errors.New("test error")
				},
				syncWebFeatureChromiumHistogramEnumValues: func(
					_ context.Context, in []gcpspanner.WebFeatureChromiumHistogramEnumValue) error {
					if len(in) != 0 {
						t.Error("expected empty slice")
					}

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
			name: "SyncWebFeatureChromiumHistogramEnumValues returns error",
			client: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys: func(_ context.Context) ([]string, error) {
					return []string{"enum-label"}, nil
				},
				upsertChromiumHistogramEnum: func(_ context.Context,
					_ gcpspanner.ChromiumHistogramEnum) (*string, error) {
					return valuePtr("enumID"), nil
				},
				syncChromiumHistogramEnumValues: func(_ context.Context,
					_ []gcpspanner.ChromiumHistogramEnumValue) error {
					return nil
				},
				getIDFromChromiumHistogramEnumValueKey: func(
					_ context.Context, _ string, _ int64) (*string, error) {
					return valuePtr("enumValueID"), nil
				},
				getIDFromFeatureKey: func(_ context.Context,
					_ *gcpspanner.FeatureIDFilter) (*string, error) {
					return valuePtr("featureID"), nil
				},
				syncWebFeatureChromiumHistogramEnumValues: func(_ context.Context,
					_ []gcpspanner.WebFeatureChromiumHistogramEnumValue) error {
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
		"canvas-2d",
		"canvas-2d-color-management",
		"http3",
		"intersection-observer-v2",
		"view-transitions",
		"text-wrap-style",
		"float16array",
		"uint8array-base64-hex",
	}
	// nolint: lll // WONTFIX: useful comment with SHA
	want := map[string]string{
		"Canvas_2d":                "canvas-2d",
		"Canvas_2dColorManagement": "canvas-2d-color-management",
		"Http3":                    "http3",
		"IntersectionObserverV2":   "intersection-observer-v2",
		"TextWrapStyle":            "text-wrap-style",
		"ViewTransitions":          "view-transitions",
		"Float16array":             "float16array",
		"Uint8arrayBase64Hex":      "uint8array-base64-hex",
	}
	got := createEnumToFeatureKeyMap(featureKeys)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("createEnumToFeatureKeyMap()\ngot:  (%+v)\nwant: (%+v)\n", got, want)
	}
}

func TestMigrateMovedFeaturesForChromiumHistograms(t *testing.T) {
	testCases := []struct {
		name                         string
		histogramsToEnumMap          map[metricdatatypes.HistogramName]map[int64]*string
		histogramsToAllFeatureKeySet map[metricdatatypes.HistogramName]map[string]metricdatatypes.HistogramEnumValue
		movedFeatures                map[string]webdxfeaturetypes.FeatureMovedData
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
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"old-feature": {RedirectTarget: "new-feature", Kind: webdxfeaturetypes.Moved},
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
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"old-feature": {RedirectTarget: "new-feature", Kind: webdxfeaturetypes.Moved},
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
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{},
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
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"old-a": {RedirectTarget: "new-a", Kind: webdxfeaturetypes.Moved},
				"old-b": {RedirectTarget: "new-b", Kind: webdxfeaturetypes.Moved},
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
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"a": {RedirectTarget: "b", Kind: webdxfeaturetypes.Moved},
			},
			expectedHistogramsToEnumMap: map[metricdatatypes.HistogramName]map[int64]*string{},
			expectedErr:                 nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateMovedFeaturesForChromiumHistograms(
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
		expected      map[string]webdxfeaturetypes.FeatureMovedData
		expectedError error
	}{
		{
			name: "Success",
			mockClient: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys:                       nil,
				upsertChromiumHistogramEnum:               nil,
				syncChromiumHistogramEnumValues:           nil,
				getIDFromChromiumHistogramEnumValueKey:    nil,
				syncWebFeatureChromiumHistogramEnumValues: nil,
				getIDFromFeatureKey:                       nil,
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
			expected: map[string]webdxfeaturetypes.FeatureMovedData{
				"feature1": {
					RedirectTarget: "new-feature1",
					Kind:           webdxfeaturetypes.Moved,
				},
				"feature2": {
					RedirectTarget: "new-feature2",
					Kind:           webdxfeaturetypes.Moved,
				},
			},
			expectedError: nil,
		},
		{
			name: "Database error",
			mockClient: &mockChromiumHistogramEnumsClient{
				fetchAllFeatureKeys:                       nil,
				upsertChromiumHistogramEnum:               nil,
				syncChromiumHistogramEnumValues:           nil,
				getIDFromChromiumHistogramEnumValueKey:    nil,
				syncWebFeatureChromiumHistogramEnumValues: nil,
				getIDFromFeatureKey:                       nil,
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
				fetchAllFeatureKeys:                       nil,
				upsertChromiumHistogramEnum:               nil,
				syncChromiumHistogramEnumValues:           nil,
				getIDFromChromiumHistogramEnumValueKey:    nil,
				syncWebFeatureChromiumHistogramEnumValues: nil,
				getIDFromFeatureKey:                       nil,
				getAllMovedWebFeatures: func(_ context.Context) ([]gcpspanner.MovedWebFeature, error) {
					return []gcpspanner.MovedWebFeature{}, nil
				},
			},
			expected:      map[string]webdxfeaturetypes.FeatureMovedData{},
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

//nolint:gocognit // Not a regularly executed test. Only for trying updates to the mojom and/or upstream data
func TestChromiumEnumsParity(t *testing.T) {
	t.Skip("Used for debugging purposes.")

	// 1. Fetch and Parse data.json from web-features
	httpClient := http.DefaultClient
	client := gh.NewClient("")
	file, err := client.DownloadFileFromRelease(t.Context(), "web-platform-dx", "web-features", httpClient, "data.json")
	if err != nil {
		t.Fatalf("failed to fetch web-features data: %v", err)
	}

	parser := data.V3Parser{}
	processedData, err := parser.Parse(file.Contents)
	if err != nil {
		t.Fatalf("failed to parse web-features data: %v", err)
	}

	// Extract all valid Feature IDs (excluding Moved/Split if necessary)
	featureKeys := make([]string, 0, len(processedData.Features.Data))
	for id := range processedData.Features.Data {
		featureKeys = append(featureKeys, id)
	}

	splitFeatureKeys := make([]string, 0, len(processedData.Features.Split))
	for id := range processedData.Features.Split {
		splitFeatureKeys = append(splitFeatureKeys, id)
	}

	movedFeatureKeys := make([]string, 0, len(processedData.Features.Moved))
	for id := range processedData.Features.Moved {
		movedFeatureKeys = append(movedFeatureKeys, id)
	}

	// 2. Generate our internal mapping using the updated logic
	generatedMap := createEnumToFeatureKeyMap(featureKeys)
	generatedSplitMap := createEnumToFeatureKeyMap(splitFeatureKeys)
	generatedMovedMap := createEnumToFeatureKeyMap(movedFeatureKeys)
	// Create a fast lookup for reverse check: FeatureID -> Label
	reverseGeneratedMap := make(map[string]string)
	for label, id := range generatedMap {
		reverseGeneratedMap[id] = label
	}

	// 3. Fetch and Parse Chromium enums.xml
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, workflow.EnumURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	xmlRespBase64Encode, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch chromium enums: %v", err)
	}
	defer xmlRespBase64Encode.Body.Close()

	enumParser := workflow.ChromiumCodesearchEnumParser{}
	enums, err := enumParser.Parse(t.Context(), xmlRespBase64Encode.Body,
		[]metricdatatypes.HistogramName{metricdatatypes.WebDXFeatureEnum})
	if err != nil {
		t.Errorf("unable to parse enums %s", err)
	}
	webDXEnum := enums[metricdatatypes.WebDXFeatureEnum]
	if len(webDXEnum) == 0 {
		t.Error("found 0 entries for the web dx enum")
	}
	// Create a lookup for Chromium labels to check reverse presence
	chromiumLabels := make(map[string]bool)
	for _, e := range webDXEnum {
		chromiumLabels[e.Label] = true
	}

	// --- BI-DIRECTIONAL CHECKS ---

	// A. FORWARD CHECK: Chromium -> WebDX (Coverage)
	// Ensures every active metric sent by Chromium is recognized by our data.
	t.Run("Forward_ChromiumToWebDX", func(t *testing.T) {
		for _, enumVal := range webDXEnum {
			label := enumVal.Label
			// TODO: Remove this switch-case once http://crrev.com/c/7595793 is merged
			switch label {
			case "Canvas2D":
				label = "Canvas_2d"
			case "Canvas2DAlpha":
				label = "Canvas_2dAlpha"
			case "Canvas2DColorManagement":
				label = "Canvas_2dColorManagement"
			case "Canvas2DDesynchronized":
				label = "Canvas_2dDesynchronized"
			case "Canvas2DWillreadfrequently":
				label = "Canvas_2dWillreadfrequently"
			case "Float16Array":
				label = "Float16array"
			case "Uint8ArrayBase64Hex":
				label = "Uint8arrayBase64Hex"
			}

			if _, exists := generatedMap[label]; !exists {
				// Check if moved or split
				if id, found := generatedSplitMap[label]; found {
					t.Logf("Info: ID %q is marked as split in web-features data.json. Consider marking it as obsolete",
						id)

					continue
				}
				if id, found := generatedMovedMap[label]; found {
					t.Logf("Info: ID %q is marked as moved in web-features data.json. "+
						"Consider renaming or marking as obsolete", id)

					continue
				}
				t.Errorf("Coverage Gap: Chromium has label %q, but no matching ID found in web-features data.json",
					label)
			}
		}
	})

	// B. REVERSE CHECK: WebDX -> Chromium (Logic)
	// Ensures our transformation logic matches the labels actually in Chromium.
	t.Run("Reverse_WebDXToChromium", func(t *testing.T) {
		for id := range processedData.Features.Data {
			predictedLabel, ok := reverseGeneratedMap[id]
			if !ok {
				t.Errorf("Logic Error: Could not generate a label for ID %q", id)

				continue
			}

			// If the predicted label isn't in Chromium, check if it exists as an OBSOLETE or DRAFT version
			if !chromiumLabels[predictedLabel] {
				if chromiumLabels["OBSOLETE_"+predictedLabel] {
					t.Logf("Info: ID %q is marked OBSOLETE in Chromium as %q", id, "OBSOLETE_"+predictedLabel)

					continue
				}
				if chromiumLabels["DRAFT_"+predictedLabel] {
					t.Logf("Info: ID %q is marked DRAFT in Chromium as %q", id, "DRAFT_"+predictedLabel)

					continue
				}
				// This might happen if a feature is in web-features but hasn't landed in Chromium enums.xml yet.
				t.Logf("Sync Warning: Predicted label %q for ID %q not found in Chromium. (May not have landed yet)",
					predictedLabel, id)
			}
		}
	})

}

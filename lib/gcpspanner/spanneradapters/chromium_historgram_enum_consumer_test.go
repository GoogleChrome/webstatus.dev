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
	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

type mockChromiumHistogramEnumsClient struct {
	upsertChromiumHistogramEnum func(context.Context,
		gcpspanner.ChromiumHistogramEnum) (*string, error)
	upsertChromiumHistogramEnumValue func(context.Context,
		gcpspanner.ChromiumHistogramEnumValue) (*string, error)
	upsertWebFeatureChromiumHistogramEnumValue func(context.Context,
		gcpspanner.WebFeatureChromiumHistogramEnumValue) error
	getIDFromFeatureKey func(context.Context, *gcpspanner.FeatureIDFilter) (*string, error)
	fetchAllFeatureKeys func(context.Context) ([]string, error)
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

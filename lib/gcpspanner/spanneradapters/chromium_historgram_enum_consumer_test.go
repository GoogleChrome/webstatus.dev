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
			name: "UpsertChromiumHistogramEnum returns error",
			client: &mockChromiumHistogramEnumsClient{
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

func TestEnumLabelToFeatureKey(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  string
	}{
		{
			name:  "Simple lowercase",
			label: "simple",
			want:  "simple",
		},
		{
			name:  "typical",
			label: "TypicalCase",
			want:  "typical-case",
		},
		{
			name:  "With numbers in the middle",
			label: "With123Numbers",
			want:  "with-123-numbers",
		},
		{
			name:  "Starting with number",
			label: "123Abc",
			want:  "123-abc",
		},
		{
			name:  "Consecutive uppercase letters",
			label: "ABCTest",
			want:  "a-b-c-test",
		},
		{
			name:  "Mixed case with numbers and consecutive uppercase",
			label: "ABC123defGHI456Jkl",
			want:  "a-b-c-123def-g-h-i-456-jkl",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := enumLabelToFeatureKey(tc.label); got != tc.want {
				t.Errorf("enumLabelToFeatureKey() = %v, want %v", got, tc.want)
			}
		})
	}
}

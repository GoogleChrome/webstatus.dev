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

package spanneradapters

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"github.com/google/go-cmp/cmp"
)

func TestMigrator_Migrate(t *testing.T) {
	type testCase[SetValueType, DataType any] struct {
		name             string
		movedFeatures    map[string]webdxfeaturetypes.FeatureMovedData
		allFeaturesSet   map[string]SetValueType
		dataToMigrate    DataType
		updateFunc       func(oldKey, newKey string, data DataType)
		expectedData     DataType
		expectedErr      error
		logShouldContain string
	}

	testCases := []testCase[struct{}, map[string]int]{
		{
			name:          "no migration needed",
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{},
			allFeaturesSet: map[string]struct{}{
				"feature1": {},
				"feature2": {},
			},
			dataToMigrate: map[string]int{
				"feature1": 1,
				"feature2": 2,
			},
			updateFunc: func(oldKey, newKey string, data map[string]int) {
				data[newKey] = data[oldKey]
				delete(data, oldKey)
			},
			expectedData: map[string]int{
				"feature1": 1,
				"feature2": 2,
			},
			logShouldContain: "",
			expectedErr:      nil,
		},
		{
			name: "successful migration",
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"feature1": {RedirectTarget: "feature3", Kind: webdxfeaturetypes.Moved},
			},
			allFeaturesSet: map[string]struct{}{
				"feature1": {},
				"feature2": {},
			},
			dataToMigrate: map[string]int{
				"feature1": 1,
				"feature2": 2,
			},
			updateFunc: func(oldKey, newKey string, data map[string]int) {
				data[newKey] = data[oldKey]
				delete(data, oldKey)
			},
			expectedData: map[string]int{
				"feature2": 2,
				"feature3": 1,
			},
			expectedErr:      nil,
			logShouldContain: "migrating feature key",
		},
		{
			name: "conflict during migration",
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"feature1": {RedirectTarget: "feature2", Kind: webdxfeaturetypes.Moved},
			},
			allFeaturesSet: map[string]struct{}{
				"feature1": {},
				"feature2": {},
			},
			dataToMigrate: map[string]int{
				"feature1": 1,
				"feature2": 2,
			},
			updateFunc: func(oldKey, newKey string, data map[string]int) {
				data[newKey] = data[oldKey]
				delete(data, oldKey)
			},
			expectedData: map[string]int{
				"feature1": 1,
				"feature2": 2,
			},
			expectedErr: ErrConflictMigratingFeatureKey,
			// Useful for GCP alerts.
			logShouldContain: "conflict migrating feature key",
		},
		{
			name: "multiple successful migrations",
			movedFeatures: map[string]webdxfeaturetypes.FeatureMovedData{
				"feature1": {RedirectTarget: "feature3", Kind: webdxfeaturetypes.Moved},
				"feature2": {RedirectTarget: "feature4", Kind: webdxfeaturetypes.Moved},
			},
			allFeaturesSet: map[string]struct{}{
				"feature1": {},
				"feature2": {},
			},
			dataToMigrate: map[string]int{
				"feature1": 1,
				"feature2": 2,
			},
			updateFunc: func(oldKey, newKey string, data map[string]int) {
				data[newKey] = data[oldKey]
				delete(data, oldKey)
			},
			expectedData: map[string]int{
				"feature3": 1,
				"feature4": 2,
			},
			logShouldContain: "migrating feature key",
			expectedErr:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))

			migrator := NewMigrator(
				tc.movedFeatures,
				tc.allFeaturesSet,
				tc.dataToMigrate,
				WithLoggerForMigrator[struct{}, map[string]int](logger),
			)

			err := migrator.Migrate(context.Background(), tc.updateFunc)

			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("unexpected error. got %v, want %v", err, tc.expectedErr)
			}

			if diff := cmp.Diff(tc.expectedData, tc.dataToMigrate); diff != "" {
				t.Errorf("unexpected data after migration (-want +got):\n%s", diff)
			}

			if tc.logShouldContain != "" && !strings.Contains(logBuf.String(), tc.logShouldContain) {
				t.Errorf("log output does not contain expected string '%s'. got: %s", tc.logShouldContain, logBuf.String())
			}
		})
	}
}

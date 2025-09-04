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
	"context"
	"errors"
	"log/slog"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

// ErrConflictMigratingFeatureKey is returned when a feature key migration would result in a conflict.
// This occurs when both the old feature key and the new feature key are present in the input data,
// indicating that the upstream data source is in an inconsistent state.
var ErrConflictMigratingFeatureKey = errors.New("conflict migrating feature key")

// Migrator is a generic helper designed to handle the migration of data associated with renamed feature keys.
// It iterates through a provided dataset, identifies features that have been moved (renamed),
// and applies a user-defined update function to migrate the data to the new feature key.
//
// SetValueType is the type of the value in the set of all features. It is not used directly but is
// required for the generic map type.
// DataType is the type of the data structure that holds the information to be migrated.
type Migrator[SetValueType, DataType any] struct {
	// AllFeaturesSet is a map representing the set of all feature keys present in the source data.
	// The key is the feature identifier (string). The value's type is generic and not used by the migrator logic.
	AllFeaturesSet map[string]SetValueType
	// MovedFeatures is a map where the key is the old (moved) feature key and the value
	// contains information about the new feature key.
	MovedFeatures map[string]web_platform_dx__web_features.FeatureMovedData
	// DataToMigrate is the actual data structure that needs to be modified based on the feature key migrations.
	DataToMigrate DataType
	// logger is an optional logger for outputting migration information. Defaults to slog.Default().
	logger *slog.Logger
}

// MigratorOption defines a function signature for configuring a Migrator instance.
type MigratorOption[SetValueType, DataType any] func(*Migrator[SetValueType, DataType])

// WithLoggerForMigrator returns a MigratorOption to set a custom logger for the Migrator.
// This allows capturing migration warnings and errors in a structured way.
func WithLoggerForMigrator[SetValueType, DataType any](logger *slog.Logger) MigratorOption[SetValueType, DataType] {
	return func(m *Migrator[SetValueType, DataType]) {
		m.logger = logger
	}
}

// NewMigrator creates and returns a new Migrator instance.
// It takes the moved features map, the set of all features from the source data, the data to be migrated,
// and optional configuration functions.
func NewMigrator[SetValueType, DataType any](
	movedFeatures map[string]web_platform_dx__web_features.FeatureMovedData,
	allFeaturesSet map[string]SetValueType,
	data DataType,
	options ...MigratorOption[SetValueType, DataType],
) *Migrator[SetValueType, DataType] {
	m := &Migrator[SetValueType, DataType]{
		AllFeaturesSet: allFeaturesSet,
		MovedFeatures:  movedFeatures,
		DataToMigrate:  data,
		logger:         slog.Default(),
	}
	for _, option := range options {
		option(m)
	}

	return m
}

// Migrate executes the feature key migration process.
// It iterates over all feature keys in the input data. If a feature key has been moved,
// it checks for conflicts. A conflict arises if the target (new) feature key already exists in the input data.
// If a conflict is found, it returns ErrConflictMigratingFeatureKey.
// If there is no conflict, it logs a warning and calls the provided `update` function
// with the old key, new key, and the data structure to be migrated.
func (m *Migrator[SetValueType, DataType]) Migrate(
	ctx context.Context, update func(oldKey, newKey string, data DataType)) error {
	for featureKey := range m.AllFeaturesSet {
		if movedFeatureData, found := m.MovedFeatures[featureKey]; found {
			if _, exists := m.AllFeaturesSet[movedFeatureData.RedirectTarget]; exists {
				m.logger.ErrorContext(ctx, "conflict migrating feature key. upstream currently using both keys",
					"old_key", featureKey,
					"new_key", movedFeatureData.RedirectTarget)

				return ErrConflictMigratingFeatureKey
			}

			m.logger.WarnContext(ctx, "migrating feature key",
				"old_key", featureKey,
				"new_key", movedFeatureData.RedirectTarget,
			)
			update(featureKey, movedFeatureData.RedirectTarget, m.DataToMigrate)
		}
	}

	return nil
}

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

package v1

import (
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes/featurestate"
)

const (
	// KindFeatureListSnapshot identifies a full state dump of features.
	KindFeatureListSnapshot = "FeatureListSnapshot"

	// VersionFeatureListSnapshot identifies v1 of the FeatureListSnapshot schema.
	VersionFeatureListSnapshot = "v1"
)

// FeatureListSnapshotV1 represents the persisted state of a search.
type FeatureListSnapshotV1 struct {
	Metadata StateMetadataV1   `json:"metadata"`
	Data     FeatureListDataV1 `json:"data"`
}

func (s FeatureListSnapshotV1) Kind() string    { return KindFeatureListSnapshot }
func (s FeatureListSnapshotV1) Version() string { return VersionFeatureListSnapshot }

type StateMetadataV1 struct {
	ID             string    `json:"id"`
	GeneratedAt    time.Time `json:"generatedAt"`
	SearchID       string    `json:"searchId"`
	QuerySignature string    `json:"querySignature"`
	EventID        string    `json:"eventId,omitempty"`
}

type FeatureListDataV1 struct {
	Features map[string]featurestate.ComparableFeature `json:"features"`
}

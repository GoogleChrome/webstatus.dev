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

package data

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
)

// Parser contains the logic to parse the JSON from the web-features Github Release.
type Parser struct{}

// V3Parser contains the logic to parse the JSON from the web-features Github Release.
type V3Parser struct{}

var ErrUnexpectedFormat = errors.New("unexpected format")

var ErrUnableToProcess = errors.New("unable to process the data")

// rawWebFeaturesJSONDataV2 is used to parse the source JSON.
// It holds the features as raw JSON messages to be processed individually.
type rawWebFeaturesJSONDataV2 struct {
	Browsers  web_platform_dx__web_features.Browsers                `json:"browsers"`
	Groups    map[string]web_platform_dx__web_features.GroupData    `json:"groups"`
	Snapshots map[string]web_platform_dx__web_features.SnapshotData `json:"snapshots"`
	Features  map[string]web_platform_dx__web_features.FeatureValue `json:"features"`
}

// rawWebFeaturesJSONDataV3 is used to parse the source JSON.
// It holds the features as raw JSON messages to be processed individually.
type rawWebFeaturesJSONDataV3 struct {
	Browsers  web_platform_dx__web_features.Browsers                `json:"browsers"`
	Groups    map[string]web_platform_dx__web_features.GroupData    `json:"groups"`
	Snapshots map[string]web_platform_dx__web_features.SnapshotData `json:"snapshots"`
	// TODO: When we move to v3, we will change Features to being json.RawMessage
	Features json.RawMessage `json:"features"`
}

// featureKindPeek is a small helper struct to find the discriminator value in V3.
type featureKindPeek struct {
	Kind string `json:"kind"`
}

// Parse expects the raw bytes for a map of string to
// https://github.com/web-platform-dx/web-features/blob/main/schemas/defs.schema.json
// The string is the feature ID.
// It will consume the readcloser and close it.
func (p Parser) Parse(in io.ReadCloser) (*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	defer in.Close()
	var source rawWebFeaturesJSONDataV2
	decoder := json.NewDecoder(in)
	err := decoder.Decode(&source)
	if err != nil {
		return nil, errors.Join(ErrUnexpectedFormat, err)
	}

	processedData := postProcess(&source)

	return processedData, nil
}

// Parse expects the raw bytes for a map of string to
// https://github.com/web-platform-dx/web-features/blob/main/schemas/defs.schema.json
// The string is the feature ID.
// It will consume the readcloser and close it.
func (p V3Parser) Parse(in io.ReadCloser) (*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	defer in.Close()
	var source rawWebFeaturesJSONDataV3
	decoder := json.NewDecoder(in)
	err := decoder.Decode(&source)
	if err != nil {
		return nil, errors.Join(ErrUnexpectedFormat, err)
	}

	processedData, err := postProcessV3(&source)
	if err != nil {
		return nil, errors.Join(ErrUnableToProcess, err)
	}

	return processedData, nil
}

func postProcess(data *rawWebFeaturesJSONDataV2) *webdxfeaturetypes.ProcessedWebFeaturesData {
	featureKinds := postProcessFeatureValue(data.Features)

	return &webdxfeaturetypes.ProcessedWebFeaturesData{
		Browsers:  data.Browsers,
		Groups:    data.Groups,
		Snapshots: data.Snapshots,
		Features:  featureKinds,
	}
}

func postProcessV3(data *rawWebFeaturesJSONDataV3) (*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	featureKinds, err := postProcessFeatureValueV3(data.Features)
	if err != nil {
		return nil, err
	}

	return &webdxfeaturetypes.ProcessedWebFeaturesData{
		Browsers:  data.Browsers,
		Groups:    data.Groups,
		Snapshots: data.Snapshots,
		Features:  featureKinds,
	}, nil
}

func postProcessFeatureValueV3(data json.RawMessage) (*webdxfeaturetypes.FeatureKinds, error) {
	featureKinds := webdxfeaturetypes.FeatureKinds{
		Data:  nil,
		Moved: nil,
		Split: nil,
	}

	featureRawMessageMap := make(map[string]json.RawMessage)

	err := json.Unmarshal(data, &featureRawMessageMap)
	if err != nil {
		return nil, err
	}

	for id, rawFeature := range featureRawMessageMap {
		// Peek inside the raw JSON to find the "kind"
		var peek featureKindPeek
		if err := json.Unmarshal(rawFeature, &peek); err != nil {
			// Skip or log features that don't have a 'kind' field
			continue
		}

		// Switch on the explicit "kind" to unmarshal into the correct type
		switch peek.Kind {
		case string(web_platform_dx__web_features.Feature):
			if featureKinds.Data == nil {
				featureKinds.Data = make(map[string]web_platform_dx__web_features.FeatureValue)
			}
			var value web_platform_dx__web_features.FeatureValue
			if err := json.Unmarshal(rawFeature, &value); err != nil {
				return nil, err
			}
			// Run your existing post-processing logic
			featureKinds.Data[id] = web_platform_dx__web_features.FeatureValue{
				Caniuse:         postProcessStringOrStringArray(value.Caniuse),
				CompatFeatures:  value.CompatFeatures,
				Description:     value.Description,
				DescriptionHTML: value.DescriptionHTML,
				Group:           postProcessStringOrStringArray(value.Group),
				Name:            value.Name,
				Snapshot:        postProcessStringOrStringArray(value.Snapshot),
				Spec:            postProcessStringOrStringArray(value.Spec),
				Status:          postProcessStatus(value.Status),
				Discouraged:     value.Discouraged,
			}

		case string(web_platform_dx__web_features.Moved):
			if featureKinds.Moved == nil {
				featureKinds.Moved = make(map[string]web_platform_dx__web_features.FeatureMovedData)
			}
			var value web_platform_dx__web_features.FeatureMovedData
			if err := json.Unmarshal(rawFeature, &value); err != nil {
				return nil, err
			}
			featureKinds.Moved[id] = value

		case string(web_platform_dx__web_features.Split):
			if featureKinds.Split == nil {
				featureKinds.Split = make(map[string]web_platform_dx__web_features.FeatureSplitData)
			}
			var value web_platform_dx__web_features.FeatureSplitData
			if err := json.Unmarshal(rawFeature, &value); err != nil {
				return nil, err
			}
			featureKinds.Split[id] = value
		}
	}

	return &featureKinds, nil
}
func postProcessFeatureValue(
	data map[string]web_platform_dx__web_features.FeatureValue) *webdxfeaturetypes.FeatureKinds {
	featureKinds := webdxfeaturetypes.FeatureKinds{
		Data:  nil,
		Moved: nil,
		Split: nil,
	}

	for id, value := range data {
		if featureKinds.Data == nil {
			featureKinds.Data = make(map[string]web_platform_dx__web_features.FeatureValue)
		}
		featureKinds.Data[id] = web_platform_dx__web_features.FeatureValue{
			Caniuse:         postProcessStringOrStringArray(value.Caniuse),
			CompatFeatures:  value.CompatFeatures,
			Description:     value.Description,
			DescriptionHTML: value.DescriptionHTML,
			Group:           postProcessStringOrStringArray(value.Group),
			Name:            value.Name,
			Snapshot:        postProcessStringOrStringArray(value.Snapshot),
			Spec:            postProcessStringOrStringArray(value.Spec),
			Status:          postProcessStatus(value.Status),
			Discouraged:     value.Discouraged,
		}
	}

	return &featureKinds
}

func postProcessStringOrStringArray(
	value *web_platform_dx__web_features.StringOrStringArray) *web_platform_dx__web_features.StringOrStringArray {
	// Do nothing for now.
	if value == nil {
		return nil
	}

	return &web_platform_dx__web_features.StringOrStringArray{
		String:      value.String,
		StringArray: value.StringArray,
	}
}

func postProcessStatus(value web_platform_dx__web_features.Status) web_platform_dx__web_features.Status {
	return web_platform_dx__web_features.Status{
		Baseline:         postProcessBaseline(value.Baseline),
		BaselineHighDate: postProcessBaselineDates(value.BaselineHighDate),
		BaselineLowDate:  postProcessBaselineDates(value.BaselineLowDate),
		Support:          postProcessBaselineSupport(value.Support),
		ByCompatKey:      nil,
	}
}

func postProcessBaselineDates(value *string) *string {
	if value == nil {
		return nil
	}
	*value = removeRangeSymbol(*value)

	return value
}

func postProcessBaseline(
	value *web_platform_dx__web_features.BaselineUnion) *web_platform_dx__web_features.BaselineUnion {
	if value == nil {
		return nil
	}

	return &web_platform_dx__web_features.BaselineUnion{
		Bool: value.Bool,
		Enum: value.Enum,
	}
}

func postProcessBaselineSupportBrowser(value *string) *string {
	if value == nil {
		return nil
	}
	*value = removeRangeSymbol(*value)

	return value
}

func postProcessBaselineSupport(
	value web_platform_dx__web_features.StatusSupport) web_platform_dx__web_features.StatusSupport {
	return web_platform_dx__web_features.StatusSupport{
		Chrome:         postProcessBaselineSupportBrowser(value.Chrome),
		ChromeAndroid:  postProcessBaselineSupportBrowser(value.ChromeAndroid),
		Edge:           postProcessBaselineSupportBrowser(value.Edge),
		Firefox:        postProcessBaselineSupportBrowser(value.Firefox),
		FirefoxAndroid: postProcessBaselineSupportBrowser(value.FirefoxAndroid),
		Safari:         postProcessBaselineSupportBrowser(value.Safari),
		SafariIos:      postProcessBaselineSupportBrowser(value.SafariIos),
	}
}

// Removes web-features range character "≤" from the string.
func removeRangeSymbol(value string) string {
	return strings.TrimPrefix(value, "≤")
}

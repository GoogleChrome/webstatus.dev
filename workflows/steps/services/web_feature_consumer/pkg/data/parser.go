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

var ErrUnexpectedFormat = errors.New("unexpected format")

var ErrUnableToProcess = errors.New("unable to process the data")

// rawWebFeaturesJSONData is used to parse the source JSON.
// It holds the features as raw JSON messages to be processed individually.
type rawWebFeaturesJSONData struct {
	Browsers map[string]web_platform_dx__web_features.BrowserData `json:"browsers"`
	Groups   map[string]web_platform_dx__web_features.GroupData   `json:"groups"`
	Features map[string]json.RawMessage                           `json:"features"`
}

// featureKindPeek is a small helper struct to find the discriminator value.
type featureKindPeek struct {
	Kind string `json:"kind"`
}

// Parse expects the raw bytes for a map of string to
// https://github.com/web-platform-dx/web-features/blob/main/schemas/defs.schema.json
// The string is the feature ID.
// It will consume the readcloser and close it.
func (p Parser) Parse(in io.ReadCloser) (*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	defer in.Close()
	var source rawWebFeaturesJSONData
	decoder := json.NewDecoder(in)
	err := decoder.Decode(&source)
	if err != nil {
		return nil, errors.Join(ErrUnexpectedFormat, err)
	}

	ret, err := postProcess(&source)
	if err != nil {
		return nil, errors.Join(ErrUnableToProcess, err)
	}

	return ret, nil
}

func postProcess(data *rawWebFeaturesJSONData) (*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	featureKinds, err := postProcessFeatureValue(data.Features)
	if err != nil {
		return nil, err
	}
	return &webdxfeaturetypes.ProcessedWebFeaturesData{
		Browsers: data.Browsers,
		Groups:   data.Groups,
		Features: *featureKinds,
	}, nil

}

func postProcessFeatureValue(data map[string]json.RawMessage) (*webdxfeaturetypes.FeatureKinds, error) {
	featureKinds := webdxfeaturetypes.FeatureKinds{
		Data:  make(map[string]web_platform_dx__web_features.FeatureData),
		Moved: make(map[string]web_platform_dx__web_features.FeatureMovedData),
		Split: make(map[string]web_platform_dx__web_features.FeatureSplitData),
	}

	for id, rawFeature := range data {
		// Peek inside the raw JSON to find the "kind"
		var peek featureKindPeek
		if err := json.Unmarshal(rawFeature, &peek); err != nil {
			// Skip or log features that don't have a 'kind' field
			continue
		}

		// Switch on the explicit "kind" to unmarshal into the correct type
		switch peek.Kind {
		case string(web_platform_dx__web_features.Data):
			var value web_platform_dx__web_features.FeatureData
			if err := json.Unmarshal(rawFeature, &value); err != nil {
				return nil, err
			}
			// Run your existing post-processing logic
			featureKinds.Data[id] = web_platform_dx__web_features.FeatureData{
				Caniuse:         postProcessStringOrStrings(value.Caniuse),
				CompatFeatures:  value.CompatFeatures,
				Description:     value.Description,
				DescriptionHTML: value.DescriptionHTML,
				Group:           postProcessStringOrStrings(value.Group),
				Name:            value.Name,
				Snapshot:        postProcessStringOrStrings(value.Snapshot),
				Spec:            postProcessStringOrStrings(value.Spec),
				Status:          postProcessStatus(value.Status),
				Discouraged:     value.Discouraged,
			}

		case string(web_platform_dx__web_features.Moved):
			var value web_platform_dx__web_features.FeatureMovedData
			if err := json.Unmarshal(rawFeature, &value); err != nil {
				return nil, err
			}
			featureKinds.Moved[id] = value

		case string(web_platform_dx__web_features.Split):
			var value web_platform_dx__web_features.FeatureSplitData
			if err := json.Unmarshal(rawFeature, &value); err != nil {
				return nil, err
			}
			featureKinds.Split[id] = value
		}
	}

	return &featureKinds, nil
}

func postProcessStringOrStrings(
	value *web_platform_dx__web_features.StringOrStrings) *web_platform_dx__web_features.StringOrStrings {
	// Do nothing for now.
	if value == nil {
		return nil
	}

	return &web_platform_dx__web_features.StringOrStrings{
		String:      value.String,
		StringArray: value.StringArray,
	}
}

func postProcessStatus(value web_platform_dx__web_features.StatusHeadlineClass) web_platform_dx__web_features.StatusHeadlineClass {
	return web_platform_dx__web_features.StatusHeadlineClass{
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
	value web_platform_dx__web_features.ByCompatKeySupport) web_platform_dx__web_features.ByCompatKeySupport {
	return web_platform_dx__web_features.ByCompatKeySupport{
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

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
)

// Parser contains the logic to parse the JSON from the web-features Github Release.
type Parser struct{}

var ErrUnexpectedFormat = errors.New("unexpected format")

// Parse expects the raw bytes for a map of string to
// https://github.com/web-platform-dx/web-features/blob/main/schemas/defs.schema.json
// The string is the feature ID.
// It will consume the readcloser and close it.
func (p Parser) Parse(in io.ReadCloser) (*web_platform_dx__web_features.FeatureData, error) {
	defer in.Close()
	var ret web_platform_dx__web_features.FeatureData
	decoder := json.NewDecoder(in)
	err := decoder.Decode(&ret)
	if err != nil {
		return nil, errors.Join(ErrUnexpectedFormat, err)
	}

	postProcess(&ret)

	return &ret, nil
}

func postProcess(data *web_platform_dx__web_features.FeatureData) {
	postProcessFeatureValue(data.Features)
}

func postProcessFeatureValue(
	data map[string]web_platform_dx__web_features.FeatureValue) {
	for id, value := range data {
		data[id] = web_platform_dx__web_features.FeatureValue{
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

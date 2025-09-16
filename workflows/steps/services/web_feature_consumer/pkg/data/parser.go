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
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features_v3"
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
	Browsers  web_platform_dx__web_features_v3.Browsers                `json:"browsers"`
	Groups    map[string]web_platform_dx__web_features_v3.GroupData    `json:"groups"`
	Snapshots map[string]web_platform_dx__web_features_v3.SnapshotData `json:"snapshots"`
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
		Browsers:  postProcessBrowsers(data.Browsers),
		Groups:    postProcessGroups(data.Groups),
		Snapshots: postProcessSnapshots(data.Snapshots),
		Features:  featureKinds,
	}
}

func postProcessV3(data *rawWebFeaturesJSONDataV3) (*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	featureKinds, err := postProcessFeatureValueV3(data.Features)
	if err != nil {
		return nil, err
	}

	return &webdxfeaturetypes.ProcessedWebFeaturesData{
		Browsers:  postProcessBrowsersV3(data.Browsers),
		Groups:    postProcessGroupsV3(data.Groups),
		Snapshots: postProcessSnapshotsV3(data.Snapshots),
		Features:  featureKinds,
	}, nil
}

func postProcessBrowsersV3(value web_platform_dx__web_features_v3.Browsers) webdxfeaturetypes.Browsers {
	return webdxfeaturetypes.Browsers{
		Chrome:         postProcessBrowserDataV3(value.Chrome),
		ChromeAndroid:  postProcessBrowserDataV3(value.ChromeAndroid),
		Edge:           postProcessBrowserDataV3(value.Edge),
		Firefox:        postProcessBrowserDataV3(value.Firefox),
		FirefoxAndroid: postProcessBrowserDataV3(value.FirefoxAndroid),
		Safari:         postProcessBrowserDataV3(value.Safari),
		SafariIos:      postProcessBrowserDataV3(value.SafariIos),
	}
}

func postProcessBrowserDataV3(value web_platform_dx__web_features_v3.BrowserData) webdxfeaturetypes.BrowserData {
	var releases []webdxfeaturetypes.Release
	if value.Releases != nil {
		releases = make([]webdxfeaturetypes.Release, len(value.Releases))
		for i, r := range value.Releases {
			releases[i] = webdxfeaturetypes.Release{
				Version: r.Version,
				Date:    r.Date,
			}
		}
	}

	return webdxfeaturetypes.BrowserData{Name: value.Name, Releases: releases}
}

func postProcessGroupsV3(
	value map[string]web_platform_dx__web_features_v3.GroupData) map[string]webdxfeaturetypes.GroupData {
	if value == nil {
		return nil
	}
	groups := make(map[string]webdxfeaturetypes.GroupData, len(value))
	for id, g := range value {
		groups[id] = webdxfeaturetypes.GroupData{
			Name:   g.Name,
			Parent: g.Parent,
		}
	}

	return groups
}

func postProcessSnapshotsV3(
	value map[string]web_platform_dx__web_features_v3.SnapshotData) map[string]webdxfeaturetypes.SnapshotData {
	if value == nil {
		return nil
	}
	snapshots := make(map[string]webdxfeaturetypes.SnapshotData, len(value))
	for id, s := range value {
		snapshots[id] = webdxfeaturetypes.SnapshotData{
			Name: s.Name,
			Spec: s.Spec,
		}
	}

	return snapshots
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
		case string(web_platform_dx__web_features_v3.Feature):
			if featureKinds.Data == nil {
				featureKinds.Data = make(map[string]webdxfeaturetypes.FeatureValue)
			}
			feature, err := processFeatureKind(rawFeature)
			if err != nil {
				return nil, err
			}
			featureKinds.Data[id] = *feature

		case string(web_platform_dx__web_features_v3.Moved):
			if featureKinds.Moved == nil {
				featureKinds.Moved = make(map[string]webdxfeaturetypes.FeatureMovedData)
			}
			moved, err := processMovedKind(rawFeature)
			if err != nil {
				return nil, err
			}
			featureKinds.Moved[id] = *moved

		case string(web_platform_dx__web_features_v3.Split):
			if featureKinds.Split == nil {
				featureKinds.Split = make(map[string]webdxfeaturetypes.FeatureSplitData)
			}
			split, err := processSplitKind(rawFeature)
			if err != nil {
				return nil, err
			}
			featureKinds.Split[id] = *split
		}
	}

	return &featureKinds, nil
}

// processFeatureKind processes a feature of kind "feature".
func processFeatureKind(rawFeature json.RawMessage) (*webdxfeaturetypes.FeatureValue, error) {
	var value web_platform_dx__web_features_v3.FeatureValue
	if err := json.Unmarshal(rawFeature, &value); err != nil {
		return nil, err
	}
	// Return an error because these values should be present. Quicktype just messes it up.
	if value.Description == nil || value.DescriptionHTML == nil || value.Name == nil || value.Status == nil {
		return nil, ErrUnexpectedFormat
	}
	feature := &webdxfeaturetypes.FeatureValue{
		Caniuse:         value.Caniuse,
		CompatFeatures:  value.CompatFeatures,
		Description:     *value.Description,
		DescriptionHTML: *value.DescriptionHTML,
		Group:           value.Group,
		Name:            *value.Name,
		Snapshot:        value.Snapshot,
		Spec:            value.Spec,
		Status:          postProcessStatusV3(*value.Status),
		Discouraged:     postProcessDiscouragedV3(value.Discouraged),
	}

	return feature, nil
}

// processMovedKind processes a feature of kind "moved".
func processMovedKind(rawFeature json.RawMessage) (*webdxfeaturetypes.FeatureMovedData, error) {
	var value web_platform_dx__web_features_v3.FeatureValue
	if err := json.Unmarshal(rawFeature, &value); err != nil {
		return nil, err
	}
	// Return an error because these values should be present. Quicktype just messes it up.
	if value.RedirectTarget == nil {
		return nil, ErrUnexpectedFormat
	}
	moved := &webdxfeaturetypes.FeatureMovedData{
		Kind:           webdxfeaturetypes.FeatureMovedDataKind(value.Kind),
		RedirectTarget: *value.RedirectTarget,
	}

	return moved, nil
}

// processSplitKind processes a feature of kind "split".
func processSplitKind(rawFeature json.RawMessage) (*webdxfeaturetypes.FeatureSplitData, error) {
	var value web_platform_dx__web_features_v3.FeatureValue
	if err := json.Unmarshal(rawFeature, &value); err != nil {
		return nil, err
	}
	// Return an error because these values should be present. Quicktype just messes it up.
	if value.RedirectTargets == nil {
		return nil, ErrUnexpectedFormat
	}
	split := &webdxfeaturetypes.FeatureSplitData{
		Kind:            webdxfeaturetypes.FeatureSplitDataKind(value.Kind),
		RedirectTargets: value.RedirectTargets,
	}

	return split, nil
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
			featureKinds.Data = make(map[string]webdxfeaturetypes.FeatureValue)
		}
		featureKinds.Data[id] = webdxfeaturetypes.FeatureValue{
			Caniuse:         postProcessStringOrStringArray(value.Caniuse),
			CompatFeatures:  value.CompatFeatures,
			Description:     value.Description,
			DescriptionHTML: value.DescriptionHTML,
			Group:           postProcessStringOrStringArray(value.Group),
			Name:            value.Name,
			Snapshot:        postProcessStringOrStringArray(value.Snapshot),
			Spec:            postProcessStringOrStringArray(value.Spec),
			Status:          postProcessStatus(value.Status), // This line is causing the error
			Discouraged:     postProcessDiscouraged(value.Discouraged),
		}
	}

	return &featureKinds
}

func postProcessStringOrStringArray(
	value *web_platform_dx__web_features.StringOrStringArray) []string {
	// Do nothing for now.
	if value == nil || (value.StringArray == nil && value.String == nil) {
		return nil
	}
	if value.String != nil {
		return []string{*value.String}
	}

	return value.StringArray
}

func postProcessDiscouraged(
	value *web_platform_dx__web_features.Discouraged) *webdxfeaturetypes.Discouraged {
	if value == nil {
		return nil
	}

	return &webdxfeaturetypes.Discouraged{
		AccordingTo:  value.AccordingTo,
		Alternatives: value.Alternatives,
	}
}

func postProcessDiscouragedV3(
	value *web_platform_dx__web_features_v3.Discouraged) *webdxfeaturetypes.Discouraged {
	if value == nil {
		return nil
	}

	return &webdxfeaturetypes.Discouraged{
		AccordingTo:  value.AccordingTo,
		Alternatives: value.Alternatives,
	}
}

func postProcessStatus(value web_platform_dx__web_features.Status) webdxfeaturetypes.Status {
	return webdxfeaturetypes.Status{
		Baseline:         postProcessBaseline(value.Baseline),
		BaselineHighDate: postProcessBaselineDates(value.BaselineHighDate),
		BaselineLowDate:  postProcessBaselineDates(value.BaselineLowDate),
		Support:          postProcessBaselineSupport(value.Support),
		ByCompatKey:      nil,
	}
}

func postProcessStatusV3(value web_platform_dx__web_features_v3.StatusHeadline) webdxfeaturetypes.Status {
	return webdxfeaturetypes.Status{
		Baseline:         postProcessBaselineV3(value.Baseline),
		BaselineHighDate: postProcessBaselineDates(value.BaselineHighDate),
		BaselineLowDate:  postProcessBaselineDates(value.BaselineLowDate),
		Support:          postProcessBaselineSupportV3(value.Support),
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

func valuePtr[T any](in T) *T { return &in }

func postProcessBaseline(
	value *web_platform_dx__web_features.BaselineUnion) *webdxfeaturetypes.BaselineUnion {
	if value == nil {
		return nil
	}
	var enum *webdxfeaturetypes.BaselineEnum
	if value.Enum != nil {
		switch *value.Enum {
		case web_platform_dx__web_features.High:
			enum = valuePtr(webdxfeaturetypes.High)
		case web_platform_dx__web_features.Low:
			enum = valuePtr(webdxfeaturetypes.Low)
		}
	}

	return &webdxfeaturetypes.BaselineUnion{
		Bool: value.Bool,
		Enum: enum,
	}
}

func postProcessBaselineV3(
	value *web_platform_dx__web_features_v3.BaselineUnion) *webdxfeaturetypes.BaselineUnion {
	if value == nil {
		return nil
	}
	var enum *webdxfeaturetypes.BaselineEnum
	if value.Enum != nil {
		switch *value.Enum {
		case web_platform_dx__web_features_v3.High:
			enum = valuePtr(webdxfeaturetypes.High)
		case web_platform_dx__web_features_v3.Low:
			enum = valuePtr(webdxfeaturetypes.Low)
		}
	}

	return &webdxfeaturetypes.BaselineUnion{
		Bool: value.Bool,
		Enum: enum,
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
	value web_platform_dx__web_features.StatusSupport) webdxfeaturetypes.StatusSupport {
	return webdxfeaturetypes.StatusSupport{
		Chrome:         postProcessBaselineSupportBrowser(value.Chrome),
		ChromeAndroid:  postProcessBaselineSupportBrowser(value.ChromeAndroid),
		Edge:           postProcessBaselineSupportBrowser(value.Edge),
		Firefox:        postProcessBaselineSupportBrowser(value.Firefox),
		FirefoxAndroid: postProcessBaselineSupportBrowser(value.FirefoxAndroid),
		Safari:         postProcessBaselineSupportBrowser(value.Safari),
		SafariIos:      postProcessBaselineSupportBrowser(value.SafariIos),
	}
}

func postProcessBaselineSupportV3(
	value web_platform_dx__web_features_v3.Support) webdxfeaturetypes.StatusSupport {
	return webdxfeaturetypes.StatusSupport{
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

func postProcessBrowsers(value web_platform_dx__web_features.Browsers) webdxfeaturetypes.Browsers {
	return webdxfeaturetypes.Browsers{
		Chrome:         postProcessBrowserData(value.Chrome),
		ChromeAndroid:  postProcessBrowserData(value.ChromeAndroid),
		Edge:           postProcessBrowserData(value.Edge),
		Firefox:        postProcessBrowserData(value.Firefox),
		FirefoxAndroid: postProcessBrowserData(value.FirefoxAndroid),
		Safari:         postProcessBrowserData(value.Safari),
		SafariIos:      postProcessBrowserData(value.SafariIos),
	}
}

func postProcessBrowserData(value web_platform_dx__web_features.BrowserData) webdxfeaturetypes.BrowserData {
	var releases []webdxfeaturetypes.Release
	if value.Releases != nil {
		releases := make([]webdxfeaturetypes.Release, len(value.Releases))
		for i, r := range value.Releases {
			releases[i] = webdxfeaturetypes.Release{
				Version: r.Version,
				Date:    r.Date,
			}
		}
	}

	return webdxfeaturetypes.BrowserData{Name: value.Name, Releases: releases}
}

func postProcessGroups(
	value map[string]web_platform_dx__web_features.GroupData) map[string]webdxfeaturetypes.GroupData {
	if value == nil {
		return nil
	}
	groups := make(map[string]webdxfeaturetypes.GroupData, len(value))
	for id, g := range value {
		groups[id] = webdxfeaturetypes.GroupData{
			Name:   g.Name,
			Parent: g.Parent,
		}
	}

	return groups
}

func postProcessSnapshots(
	value map[string]web_platform_dx__web_features.SnapshotData) map[string]webdxfeaturetypes.SnapshotData {
	if value == nil {
		return nil
	}
	snapshots := make(map[string]webdxfeaturetypes.SnapshotData, len(value))
	for id, s := range value {
		snapshots[id] = webdxfeaturetypes.SnapshotData{
			Name: s.Name,
			Spec: s.Spec,
		}
	}

	return snapshots
}

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

package workflow

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"log/slog"
	"strings"

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

// ChromiumCodesearchEnumFetcher parses the enums from Chromium code search.
// It expects the base64 encoded stream from ChromiumCodesearchEnumFetcher.
type ChromiumCodesearchEnumParser struct{}

type HistogramConfiguration struct {
	XMLName xml.Name       `xml:"histogram-configuration"`
	Enums   HistogramEnums `xml:"enums"`
}

type HistogramEnums struct {
	XMLName    xml.Name    `xml:"enums"`
	Histograms []Histogram `xml:"enum"`
}

type Histogram struct {
	XMLName xml.Name `xml:"enum"`
	Name    string   `xml:"name,attr"`

	Values []HistogramInt `xml:"int"`
}

type HistogramInt struct {
	XMLName xml.Name `xml:"int"`
	Value   int64    `xml:"value,attr"`
	Label   string   `xml:"label,attr"`
}

var (
	errMissingHistogram = errors.New("unable to find histogram")
)

// Inspired by the get_template_data method on the HistogramsHandler class
// https://github.com/GoogleChrome/chromium-dashboard/blob/main/internals/fetchmetrics.py
func (p ChromiumCodesearchEnumParser) Parse(
	ctx context.Context,
	encodedData io.ReadCloser,
	filteredHistogramEnums []metricdatatypes.HistogramName) (metricdatatypes.HistogramMapping, error) {
	defer encodedData.Close()
	rawDataDecoder := base64.NewDecoder(base64.StdEncoding, encodedData)
	xmlDataDecoder := xml.NewDecoder(rawDataDecoder)
	var config HistogramConfiguration
	err := xmlDataDecoder.Decode(&config)
	if err != nil {
		slog.ErrorContext(ctx, "error decoding xml", "error", err)

		return nil, err
	}
	m := make(metricdatatypes.HistogramMapping)
	for _, filteredHistogramEnum := range filteredHistogramEnums {
		m[filteredHistogramEnum] = nil
	}
	for _, histogram := range config.Enums.Histograms {

		histogramName := metricdatatypes.HistogramName(histogram.Name)
		_, found := m[histogramName]
		if !found {
			continue
		}
		var bucketIDToSkip *int64
		switch histogramName {
		case metricdatatypes.WebDXFeatureEnum:
			bucketIDToSkip = valuePtr[int64](0)
		}
		enums := []metricdatatypes.HistogramEnumValue{}
		for _, value := range histogram.Values {

			if bucketIDToSkip != nil && *bucketIDToSkip == value.Value {
				continue
			}
			// Skip labels with DRAFT_ prefix
			if strings.HasPrefix(value.Label, "DRAFT_") {
				continue
			}

			enums = append(enums, metricdatatypes.HistogramEnumValue{
				Value: value.Value,
				Label: value.Label,
			})
		}
		m[metricdatatypes.HistogramName(histogram.Name)] = enums
	}

	for key, value := range m {
		if len(value) == 0 {
			slog.ErrorContext(ctx, "unable to find histogram", "name", key)

			return nil, errMissingHistogram
		}
	}

	return m, nil
}

func valuePtr[T any](in T) *T { return &in }

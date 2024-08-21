package workflow

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"log/slog"

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
		enums := []metricdatatypes.HistogramEnumInfo{}
		for _, value := range histogram.Values {

			if bucketIDToSkip != nil && *bucketIDToSkip == value.Value {
				continue
			}
			enums = append(enums, metricdatatypes.HistogramEnumInfo{
				BucketID: value.Value,
				Label:    value.Label,
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

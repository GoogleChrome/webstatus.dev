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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strconv"

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

var (
	errMissingXSSIPrefix  = errors.New("missing xssi prefix")
	errUnexpectedBucketID = errors.New("unable to parse bucket id")
	errInvalidJSON        = errors.New("unable to decode json payload")
)

type XSSIMetricsParser struct{}

func removeXSSIPrefix(body io.ReadCloser) (io.ReadCloser, error) {
	var buf bytes.Buffer

	// Read and discard until newline
	_, err := buf.ReadFrom(body)
	if err != nil {
		return nil, err
	}

	// Get the remaining data from the buffer
	remainingData := buf.Bytes()

	// Find the index of the first newline
	newlineIndex := bytes.IndexByte(remainingData, '\n')
	if newlineIndex == -1 {
		return nil, errMissingXSSIPrefix
	}

	// Discard up to and including the newline
	remainingData = remainingData[newlineIndex+1:]

	// Create a new ReadCloser from the remaining data
	newBody := io.NopCloser(bytes.NewReader(remainingData))

	return newBody, nil
}

type RField map[string]metricdatatypes.BucketDataMetric
type JSONPayload struct {
	R RField `json:"r,omitempty"`
}

func (p XSSIMetricsParser) Parse(ctx context.Context, data io.ReadCloser) (metricdatatypes.BucketDataMetrics, error) {
	defer data.Close()

	jsonBody, err := removeXSSIPrefix(data)
	if err != nil {
		slog.ErrorContext(ctx, "unable to detect and remove xssi prefix", "error", err)

		return nil, err
	}
	defer jsonBody.Close()

	jsonDecoder := json.NewDecoder(jsonBody)

	var parsedData JSONPayload
	err = jsonDecoder.Decode(&parsedData)
	if err != nil {
		slog.ErrorContext(ctx, "unable to decode json payload", "error", err)

		return nil, errors.Join(err, errInvalidJSON)
	}

	ret := make(metricdatatypes.BucketDataMetrics, len(parsedData.R))
	for keyStr, data := range parsedData.R {
		keyInt, err := strconv.ParseInt(keyStr, 10, 64)
		if err != nil {
			slog.ErrorContext(ctx, "unable to parse bucket id", "error", err, "bucket", keyStr)

			return nil, errors.Join(errUnexpectedBucketID, err)
		}
		ret[keyInt] = data
	}

	return ret, nil
}

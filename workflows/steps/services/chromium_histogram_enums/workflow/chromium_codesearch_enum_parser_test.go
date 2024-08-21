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
	"encoding/base64"
	"errors"
	"io"
	"reflect"
	"strconv"
	"testing"

	"github.com/GoogleChrome/webstatus.dev/lib/metricdatatypes"
)

func TestChromiumCodesearchEnumParser_Parse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		encodedData    string
		filteredEnums  []metricdatatypes.HistogramName
		expectedOutput metricdatatypes.HistogramMapping
		expectedError  error
	}{
		{
			name: "Successful Parsing with Filtering",
			encodedData: base64.StdEncoding.EncodeToString([]byte(`
                <histogram-configuration>
                    <enums>
                        <enum name="WebDXFeatureObserver">
                            <int value="0" label="PageVisits"/>
                            <int value="1" label="CompressionStreams"/>
                            <int value="2" label="ViewTransitions"/>
                        </enum>
                        <enum name="OtherEnum">
                            <int value="10" label="SomeValue"/>
                        </enum>
                    </enums>
                </histogram-configuration>
            `)),
			filteredEnums: []metricdatatypes.HistogramName{metricdatatypes.WebDXFeatureEnum},
			expectedOutput: metricdatatypes.HistogramMapping{
				metricdatatypes.WebDXFeatureEnum: []metricdatatypes.HistogramEnumInfo{
					{BucketID: 1, Label: "CompressionStreams"},
					{BucketID: 2, Label: "ViewTransitions"},
				},
			},
			expectedError: nil,
		},
		{
			name:        "Empty Input",
			encodedData: "",
			filteredEnums: []metricdatatypes.HistogramName{
				metricdatatypes.WebDXFeatureEnum,
			},
			expectedOutput: nil,
			expectedError:  io.EOF,
		},
		{
			name: "Missing Histogram",
			encodedData: base64.StdEncoding.EncodeToString([]byte(`
                <histogram-configuration>
                    <enums>
                        <enum name="SomeOtherEnum">
                            <int value="5" label="SomeLabel"/>
                        </enum>
                    </enums>
                </histogram-configuration>
            `)),
			filteredEnums: []metricdatatypes.HistogramName{
				metricdatatypes.WebDXFeatureEnum,
			},
			expectedOutput: nil,
			expectedError:  errMissingHistogram,
		},
		{
			name: "Invalid XML",
			encodedData: base64.StdEncoding.EncodeToString([]byte(`
                <histogram-configuration>
                    <enums>
                        <enum name="WebDXFeatureObserver">
                            <int value="invalid" label="PageVisits"/>
                        </enum>
                    </enums>
                </histogram-configuration>
            `)),
			filteredEnums: []metricdatatypes.HistogramName{
				metricdatatypes.WebDXFeatureEnum,
			},
			expectedOutput: nil,
			expectedError:  strconv.ErrSyntax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := ChromiumCodesearchEnumParser{}
			encodedDataReader := io.NopCloser(bytes.NewReader([]byte(tt.encodedData)))
			output, err := parser.Parse(ctx, encodedDataReader, tt.filteredEnums)

			if tt.expectedError != nil {
				if !errors.Is(err, tt.expectedError) {
					t.Errorf("Expected error '%v', but got '%v'", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !reflect.DeepEqual(output, tt.expectedOutput) {
					t.Errorf("Output mismatch:\nExpected: %v\nGot: %v", tt.expectedOutput, output)
				}
			}
		})
	}
}

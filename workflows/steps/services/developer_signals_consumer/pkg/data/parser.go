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

package data

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/GoogleChrome/webstatus.dev/lib/developersignaltypes"
)

type rawFeatureDeveloperSignalJSON struct {
	URL   string `json:"url"`
	Votes int64  `json:"votes"`
}

type rawFeatureDeveloperSignalsJSON map[string]rawFeatureDeveloperSignalJSON

var ErrUnexpectedFormat = errors.New("unexpected format")

type Parser struct{}

func (p Parser) Parse(in io.ReadCloser) (*developersignaltypes.FeatureDeveloperSignals, error) {
	defer in.Close()
	var source rawFeatureDeveloperSignalsJSON
	decoder := json.NewDecoder(in)
	err := decoder.Decode(&source)
	if err != nil {
		return nil, errors.Join(ErrUnexpectedFormat, err)
	}

	return p.postProcess(&source), nil
}

func (p Parser) postProcess(data *rawFeatureDeveloperSignalsJSON) *developersignaltypes.FeatureDeveloperSignals {
	ret := make(developersignaltypes.FeatureDeveloperSignals, len(*data))
	for id, value := range *data {
		ret[id] = developersignaltypes.FeatureDeveloperSignal{
			Link:    value.URL,
			Upvotes: value.Votes,
		}
	}

	return &ret
}

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
	"io"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
)

type Parser struct{}

// Parse expects the raw bytes for a map of string to
// https://github.com/web-platform-dx/web-features/blob/main/schemas/defs.schema.json
// The string is the feature ID.
// It will consume the readcloser and close it.
func (p Parser) Parse(in io.ReadCloser) (map[string]web_platform_dx__web_features.FeatureData, error) {
	defer in.Close()
	var ret map[string]web_platform_dx__web_features.FeatureData
	decoder := json.NewDecoder(in)
	err := decoder.Decode(&ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

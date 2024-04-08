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

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/mdn__browser_compat_data"
)

// Parser contains the logic to parse the JSON from the bcd Github Release.
type Parser struct{}

// BCDData embeds the schema for BrowserData.
// It follows the schema found at
// https://github.com/mdn/browser-compat-data/blob/main/schemas/browsers.schema.json
type BCDData struct {
	mdn__browser_compat_data.BrowserData
}

var ErrUnexpectedFormat = errors.New("unexpected format")

// Parse attempts to parse raw byes in the BCDData.
// It will consume the readcloser and close it.
func (p Parser) Parse(in io.ReadCloser) (*BCDData, error) {
	defer in.Close()
	var ret BCDData
	decoder := json.NewDecoder(in)
	err := decoder.Decode(&ret)
	if err != nil {
		return nil, errors.Join(ErrUnexpectedFormat, err)
	}

	return &ret, nil
}

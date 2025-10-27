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
	"errors"
	"io"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features_mappings"
	"github.com/GoogleChrome/webstatus.dev/lib/webfeaturesmappingtypes"
)

var ErrUnexpectedFormat = errors.New("unexpected format")

type Parser struct{}

func (p Parser) Parse(in io.ReadCloser) (webfeaturesmappingtypes.WebFeaturesMappings, error) {
	defer in.Close()
	body, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return nil, ErrUnexpectedFormat
	}

	mappings, err := web_platform_dx__web_features_mappings.UnmarshalWebFeaturesMappings(body)
	if err != nil {
		return nil, errors.Join(ErrUnexpectedFormat, err)
	}

	return postProcess(&mappings), nil
}

func postProcess(
	mappings *web_platform_dx__web_features_mappings.WebFeaturesMappings,
) webfeaturesmappingtypes.WebFeaturesMappings {
	ret := make(webfeaturesmappingtypes.WebFeaturesMappings, len(*mappings))
	for featureID, mapping := range *mappings {
		var standardsPositions []webfeaturesmappingtypes.StandardsPosition
		if mapping.StandardsPositions != nil {
			standardsPositions = make([]webfeaturesmappingtypes.StandardsPosition, len(mapping.StandardsPositions))
			for i, sp := range mapping.StandardsPositions {
				var concerns []string
				if sp.Concerns != nil {
					concerns = make([]string, len(sp.Concerns))
					for j, c := range sp.Concerns {
						concerns[j] = string(c)
					}
				}
				standardsPositions[i] = webfeaturesmappingtypes.StandardsPosition{
					Vendor:   string(sp.Vendor),
					Position: string(sp.Position),
					URL:      sp.URL,
					Concerns: concerns,
				}
			}
		}
		ret[featureID] = webfeaturesmappingtypes.FeatureMapping{
			StandardsPositions: standardsPositions,
		}
	}

	return ret
}

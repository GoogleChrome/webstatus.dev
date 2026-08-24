// Copyright 2026 Google LLC
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

package codescan

import (
	"strings"
	"testing"
)

func TestParseBrowserslistConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fileName     string
		content      string
		wantTarget   string
		wantWarnings int
		checkWarning string
	}{
		{
			name:         ".browserslistrc with baseline newly available",
			fileName:     ".browserslistrc",
			content:      "# Browserslist configuration\n> 0.5%\nlast 2 versions\nnot dead\nbaseline newly available\n",
			wantTarget:   TargetNewly,
			wantWarnings: 0,
			checkWarning: "",
		},
		{
			name:         ".browserslistrc with baseline widely available",
			fileName:     ".browserslistrc",
			content:      "baseline widely available\n",
			wantTarget:   TargetWidely,
			wantWarnings: 0,
			checkWarning: "",
		},
		{
			name:         ".browserslistrc with section headers",
			fileName:     ".browserslistrc",
			content:      "[production]\nbaseline newly available\n[development]\nlast 1 chrome version\n",
			wantTarget:   TargetNewly,
			wantWarnings: 0,
			checkWarning: "",
		},
		{
			name:         ".browserslistrc with unsupported with downstream suffix",
			fileName:     ".browserslistrc",
			content:      "baseline widely available with downstream chrome\n",
			wantTarget:   "", // No default applied
			wantWarnings: 1,
			checkWarning: "unsupported browserslist baseline query",
		},
		{
			name:         ".browserslistrc with unsupported including kaios suffix",
			fileName:     ".browserslistrc",
			content:      "baseline newly available including kaios\n",
			wantTarget:   "",
			wantWarnings: 1,
			checkWarning: "unsupported browserslist baseline query",
		},
		{
			name:         ".browserslistrc with unsupported date cutoff",
			fileName:     ".browserslistrc",
			content:      "baseline widely available on 2024-01-01\n",
			wantTarget:   "",
			wantWarnings: 1,
			checkWarning: "unsupported browserslist baseline query",
		},
		{
			name:         ".browserslistrc with unsupported year checkpoint",
			fileName:     ".browserslistrc",
			content:      "baseline 2022\n",
			wantTarget:   "",
			wantWarnings: 1,
			checkWarning: "unsupported browserslist baseline query",
		},
		{
			name:         "package.json with array containing baseline newly available",
			fileName:     "package.json",
			content:      `{"name": "my-app", "browserslist": ["> 0.2%", "baseline newly available", "not dead"]}`,
			wantTarget:   TargetNewly,
			wantWarnings: 0,
			checkWarning: "",
		},
		{
			name:         "package.json with string baseline widely available",
			fileName:     "package.json",
			content:      `{"name": "my-app", "browserslist": "baseline widely available"}`,
			wantTarget:   TargetWidely,
			wantWarnings: 0,
			checkWarning: "",
		},
		{
			name:     "package.json with environment map",
			fileName: "package.json",
			content: `{"browserslist": {` +
				`"production": ["baseline newly available"], ` +
				`"development": ["last 1 chrome version"]}}`,
			wantTarget:   TargetNewly,
			wantWarnings: 0,
			checkWarning: "",
		},
		{
			name:         "package.json without browserslist",
			fileName:     "package.json",
			content:      `{"name": "my-app", "version": "1.0.0"}`,
			wantTarget:   "",
			wantWarnings: 0,
			checkWarning: "",
		},
		{
			name:         "package.json with unsupported baseline query",
			fileName:     "package.json",
			content:      `{"browserslist": ["baseline 2024"]}`,
			wantTarget:   "",
			wantWarnings: 1,
			checkWarning: "unsupported browserslist baseline query",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			target, warnings, err := ParseBrowserslistConfig([]byte(tc.content), tc.fileName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if target != tc.wantTarget {
				t.Errorf("target = %q, want %q", target, tc.wantTarget)
			}
			if len(warnings) != tc.wantWarnings {
				t.Errorf("got %d warnings, want %d (warnings: %v)", len(warnings), tc.wantWarnings, warnings)
			}
			if tc.checkWarning != "" && len(warnings) > 0 {
				if !strings.Contains(warnings[0], tc.checkWarning) {
					t.Errorf("warning %q does not contain %q", warnings[0], tc.checkWarning)
				}
			}
		})
	}
}

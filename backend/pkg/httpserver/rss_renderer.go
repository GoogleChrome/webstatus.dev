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

package httpserver

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

const rssItemTemplate = `
<div>
    <p>{{.SummaryText}}</p>
    {{if .Added}}
    <h4>Features Added</h4>
    <ul>
        {{range .Added}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Removed}}
    <h4>Features Removed</h4>
    <ul>
        {{range .Removed}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Other}}
    <h4>Other Updates</h4>
    <ul>
        {{range .Other}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Truncated}}
    <p><em>Note: This summary has been truncated. View full details on the site.</em></p>
    {{end}}
</div>
`

type RSSItemData struct {
	SummaryText string
	Added       []string
	Removed     []string
	Other       []string
	Truncated   bool
}

type RSSRenderer struct {
	tmpl *template.Template
}

// NewRSSRenderer initializes the renderer and parses the template at startup.
func NewRSSRenderer() *RSSRenderer {
	tmpl := template.Must(template.New("rss_item").Parse(rssItemTemplate))

	return &RSSRenderer{tmpl: tmpl}
}

func (r *RSSRenderer) RenderRSSDescription(summary workertypes.EventSummary) (string, error) {
	data := RSSItemData{
		SummaryText: summary.Text,
		Truncated:   summary.Truncated,
		Added:       []string{},
		Removed:     []string{},
		Other:       []string{},
	}

	// Map highlights to categories using Enums
	for _, h := range summary.Highlights {
		switch h.Type {
		case workertypes.SummaryHighlightTypeAdded:
			data.Added = append(data.Added, h.FeatureName)
		case workertypes.SummaryHighlightTypeRemoved:
			data.Removed = append(data.Removed, h.FeatureName)
		case workertypes.SummaryHighlightTypeChanged,
			workertypes.SummaryHighlightTypeMoved,
			workertypes.SummaryHighlightTypeSplit,
			workertypes.SummaryHighlightTypeDeleted:
			data.Other = append(data.Other, fmt.Sprintf("%s (%s)", h.FeatureName, h.Type))
		}
	}

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

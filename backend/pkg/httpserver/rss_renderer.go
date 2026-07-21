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
	"html/template"
)

const rssItemTemplate = `
<div>
    <p>{{.SummaryText}}</p>
    {{if .ResolvedQueryErrors}}
    <h4>Query Recovered</h4>
    <ul>
        {{range .ResolvedQueryErrors}}
        <li>Tracking resumed cleanly from your baseline. Resolved issue: {{.}}</li>
        {{end}}
    </ul>
    {{end}}
    {{if .QueryErrors}}
    <h4>Query Errors</h4>
    <ul>
        {{range .QueryErrors}}
        <li>{{.}}</li>
        {{end}}
    </ul>
    {{end}}
    {{if .Added}}
    <h4>Features Added</h4>
    <ul>
        {{range .Added}}
        <li>{{.}}</li>
        {{end}}
    </ul>
    {{end}}
    {{if .Removed}}
    <h4>Features Removed</h4>
    <ul>
        {{range .Removed}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Changed}}
    <h4>Features Changed</h4>
    <ul>
        {{range .Changed}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Moved}}
    <h4>Features Moved/Renamed</h4>
    <ul>
        {{range .Moved}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Split}}
    <h4>Features Split</h4>
    <ul>
        {{range .Split}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Deleted}}
    <h4>Features Deleted</h4>
    <ul>
        {{range .Deleted}}<li>{{.}}</li>{{end}}
    </ul>
    {{end}}
    {{if .Truncated}}
    <p><em>Note: This summary has been truncated. View full details on the site.</em></p>
    {{end}}
</div>
`

type RSSItemData struct {
	SummaryText         string
	Added               []string
	Removed             []string
	Changed             []string
	Moved               []string
	Split               []string
	Deleted             []string
	QueryErrors         []string
	ResolvedQueryErrors []string
	Truncated           bool
}

type RSSRenderer struct {
	tmpl *template.Template
}

// NewRSSRenderer initializes the renderer and parses the template at startup.
func NewRSSRenderer() *RSSRenderer {
	tmpl := template.Must(template.New("rss_item").Parse(rssItemTemplate))

	return &RSSRenderer{tmpl: tmpl}
}

func (r *RSSRenderer) RenderRSSDescription(data RSSItemData) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

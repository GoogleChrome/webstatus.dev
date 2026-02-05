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

package digest

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"slices"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

// HTMLRenderer implements the email_handler.TemplateRenderer interface.
type HTMLRenderer struct {
	tmpl             *template.Template
	webStatusBaseURL string
}

// NewHTMLRenderer creates a new renderer with the default template.
func NewHTMLRenderer(webStatusBaseURL string) (*HTMLRenderer, error) {
	r := &HTMLRenderer{
		webStatusBaseURL: webStatusBaseURL,
		tmpl:             nil,
	}

	// Register helper functions for the template
	funcMap := template.FuncMap{
		"toLower":              strings.ToLower,
		"browserLogoURL":       r.browserLogoURL,
		"browserDisplayName":   r.browserDisplayName,
		"statusLogoURL":        r.statusLogoURL,
		"dict":                 dict,
		"list":                 list,
		"append":               appendList,
		"formatDate":           formatDate,
		"formatBrowserStatus":  r.formatBrowserStatus,
		"formatBaselineStatus": r.formatBaselineStatus,
		"badgeBackgroundColor": badgeBackgroundColor,
		"sortedBrowserChanges": r.sortedBrowserChanges,
	}

	// Parse both the components and the main template
	tmpl, err := template.New("email").Funcs(funcMap).Parse(
		EmailStyles + EmailComponents + defaultEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email templates: %w", err)
	}
	r.tmpl = tmpl

	return r, nil
}

// dict helper function to creating maps in templates.
func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call: odd number of arguments")
	}
	dict := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}

	return dict, nil
}

func list(values ...any) []any {
	return values
}

func appendList(l []any, v any) []any {
	return append(l, v)
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.Format("2006-01-02")
}

func (r *HTMLRenderer) formatBrowserStatus(status workertypes.BrowserStatus) string {
	switch status {
	case workertypes.BrowserStatusAvailable:
		return "Available"
	case workertypes.BrowserStatusUnavailable:
		return "Unavailable"
	case workertypes.BrowserStatusUnknown:
		break
	}

	return "Unknown"
}

func (r *HTMLRenderer) formatBaselineStatus(status workertypes.BaselineStatus) string {
	switch status {
	case workertypes.BaselineStatusLimited:
		return "Limited"
	case workertypes.BaselineStatusNewly:
		return "Newly"
	case workertypes.BaselineStatusWidely:
		return "Widely"
	case workertypes.BaselineStatusUnknown:
		break
	}

	return "Unknown"
}

func badgeBackgroundColor(title string) string {
	switch title {
	case "Added":
		return "#E6F4EA" // Green
	case "Removed":
		return "#E4E4E7" // Neutral Gray
	case "Deleted":
		return "#FCE8E6" // Red
	default:
		return "#E8F0FE" // Default: Moved/Split (Blue-ish)
	}
}

// templateData is the struct passed to the HTML template.
type BrowserChangeRenderData struct {
	BrowserName workertypes.BrowserName
	Change      *workertypes.Change[workertypes.BrowserValue]
	FeatureName string
	FeatureID   string
	Type        workertypes.SummaryHighlightType
}

type templateData struct {
	Subject                   string
	FullSubject               string
	Query                     string
	SummaryText               string
	BaselineNewlyChanges      []workertypes.SummaryHighlight
	BaselineWidelyChanges     []workertypes.SummaryHighlight
	BaselineRegressionChanges []workertypes.SummaryHighlight
	AllBrowserChanges         []BrowserChangeRenderData
	AddedFeatures             []workertypes.SummaryHighlight
	RemovedFeatures           []workertypes.SummaryHighlight
	DeletedFeatures           []workertypes.SummaryHighlight
	MovedFeatures             []workertypes.SummaryHighlight
	SplitFeatures             []workertypes.SummaryHighlight
	Truncated                 bool
	BaseURL                   string
	UnsubscribeURL            string
}

// RenderDigest processes the delivery job and returns the subject and HTML body.
func (r *HTMLRenderer) RenderDigest(job workertypes.IncomingEmailDeliveryJob) (string, string, error) {
	// 1. Generate Subjects
	subject := r.generateSubject(job.Metadata.Frequency, job.Metadata.Query, true)
	fullSubject := r.generateSubject(job.Metadata.Frequency, job.Metadata.Query, false)

	// 2. Prepare Template Data using the visitor
	generator := new(templateDataGenerator)
	generator.job = job
	generator.baseURL = r.webStatusBaseURL
	generator.subject = subject
	generator.fullSubject = fullSubject

	if err := workertypes.ParseEventSummary(job.SummaryRaw, generator); err != nil {
		return "", "", fmt.Errorf("failed to parse event summary: %w", err)
	}

	// 3. Render Body
	var body bytes.Buffer
	if err := r.tmpl.Execute(&body, generator.data); err != nil {
		return "", "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return subject, body.String(), nil
}

// templateDataGenerator implements workertypes.SummaryVisitor to prepare the data for the template.
type templateDataGenerator struct {
	job         workertypes.IncomingEmailDeliveryJob
	subject     string
	fullSubject string
	baseURL     string
	data        templateData
}

// VisitV1 is called when a V1 summary is parsed.
func (g *templateDataGenerator) VisitV1(summary workertypes.EventSummary) error {
	g.data = templateData{
		Subject:     g.subject,
		FullSubject: g.fullSubject,
		Query:       g.job.Metadata.Query,
		SummaryText: summary.Text,
		Truncated:   summary.Truncated,
		BaseURL:     g.baseURL,
		UnsubscribeURL: fmt.Sprintf("%s/settings/subscriptions?unsubscribe=%s",
			g.baseURL, g.job.SubscriptionID),
		BaselineNewlyChanges:      nil,
		BaselineWidelyChanges:     nil,
		BaselineRegressionChanges: nil,
		AllBrowserChanges:         nil,
		AddedFeatures:             nil,
		RemovedFeatures:           nil,
		DeletedFeatures:           nil,
		SplitFeatures:             nil,
		MovedFeatures:             nil,
	}
	// 2. Filter Content (Content Filtering)
	// We only show highlights that match the user's specific triggers.
	filteredHighlights := filterHighlights(summary.Highlights, g.job.Triggers)
	if len(filteredHighlights) != 0 {
		// As long as we have some filtered highlights, override it.
		// This should be the common case unless there's some logic error.
		summary.Highlights = filteredHighlights
	}

	g.categorizeHighlights(summary.Highlights)

	return nil
}

func (g *templateDataGenerator) categorizeHighlights(highlights []workertypes.SummaryHighlight) {
	for _, highlight := range highlights {
		g.processHighlight(highlight)
	}
}

func (g *templateDataGenerator) processHighlight(highlight workertypes.SummaryHighlight) {
	g.routeHighlightToCategory(highlight)
}

func (g *templateDataGenerator) routeHighlightToCategory(highlight workertypes.SummaryHighlight) {
	switch highlight.Type {
	case workertypes.SummaryHighlightTypeMoved:
		g.data.MovedFeatures = append(g.data.MovedFeatures, highlight)
	case workertypes.SummaryHighlightTypeSplit:
		g.data.SplitFeatures = append(g.data.SplitFeatures, highlight)
	case workertypes.SummaryHighlightTypeAdded:
		g.data.AddedFeatures = append(g.data.AddedFeatures, highlight)
	case workertypes.SummaryHighlightTypeRemoved:
		// Promotion Strategy:
		// If a removed feature has significant changes (Baseline or Browser),
		// we treat it as a "Change" so it appears in the specific sections (e.g. Baseline Newly)
		// rather than the generic "Removed" list.
		if highlight.BaselineChange != nil || len(highlight.BrowserChanges) > 0 {
			g.processChangedData(highlight)
		} else {
			g.data.RemovedFeatures = append(g.data.RemovedFeatures, highlight)
		}
	case workertypes.SummaryHighlightTypeDeleted:
		g.data.DeletedFeatures = append(g.data.DeletedFeatures, highlight)
	case workertypes.SummaryHighlightTypeChanged:
		g.processChangedData(highlight)
	}
}

func (g *templateDataGenerator) processChangedData(highlight workertypes.SummaryHighlight) {
	// Consolidate browser changes into their own list
	if len(highlight.BrowserChanges) > 0 {
		// Sort keys to ensure deterministic order in the AllBrowserChanges list
		browsers := make([]workertypes.BrowserName, 0, len(highlight.BrowserChanges))
		for b := range highlight.BrowserChanges {
			browsers = append(browsers, b)
		}

		slices.Sort(browsers)

		for _, browserName := range browsers {
			change := highlight.BrowserChanges[browserName]
			// If a feature regresses AND loses a browser impl, it will be in two sections.
			if change == nil {
				continue
			}
			g.data.AllBrowserChanges = append(g.data.AllBrowserChanges, BrowserChangeRenderData{
				BrowserName: browserName,
				Change:      change,
				FeatureName: highlight.FeatureName,
				FeatureID:   highlight.FeatureID,
				Type:        highlight.Type,
			})
		}
	}

	if highlight.BaselineChange != nil {
		g.processBaselineChange(highlight)
	}
}

func (g *templateDataGenerator) processBaselineChange(highlight workertypes.SummaryHighlight) {
	switch highlight.BaselineChange.To.Status {
	case workertypes.BaselineStatusNewly:
		g.data.BaselineNewlyChanges = append(g.data.BaselineNewlyChanges, highlight)
	case workertypes.BaselineStatusWidely:
		g.data.BaselineWidelyChanges = append(g.data.BaselineWidelyChanges, highlight)
	case workertypes.BaselineStatusLimited:
		g.data.BaselineRegressionChanges = append(g.data.BaselineRegressionChanges, highlight)
	case workertypes.BaselineStatusUnknown:
		// Do nothing
	}
}

func filterHighlights(
	highlights []workertypes.SummaryHighlight, triggers []workertypes.JobTrigger) []workertypes.SummaryHighlight {
	// If no triggers are specified (e.g. legacy or "all"), return everything.
	if len(triggers) == 0 {
		return highlights
	}

	var filtered []workertypes.SummaryHighlight
	for _, h := range highlights {
		matched := false
		for _, t := range triggers {
			if h.MatchesTrigger(t) {
				matched = true

				break
			}
		}
		if matched {
			filtered = append(filtered, h)
		}
	}

	return filtered
}

func (r *HTMLRenderer) generateSubject(
	frequency workertypes.JobFrequency, query string, truncate bool) string {
	prefix := "Update:"
	switch frequency {
	case workertypes.FrequencyWeekly:
		prefix = "Weekly digest:"
	case workertypes.FrequencyMonthly:
		prefix = "Monthly digest:"
	case workertypes.FrequencyImmediate:
		// Do nothing
	case workertypes.FrequencyUnknown:
		// Do nothing
	}

	displayQuery := query
	if truncate && len(displayQuery) > 50 {
		displayQuery = displayQuery[:47] + "..."
	}

	return fmt.Sprintf("%s %s", prefix, displayQuery)
}

// browserToString helps handle the any passed from templates which could be
// string or workertypes.BrowserName.
func (r *HTMLRenderer) browserToString(browser any) string {
	switch v := browser.(type) {
	case string:
		return v
	case workertypes.BrowserName:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// browserLogoURL returns the URL for the browser logo.
// Maps mobile browsers to their desktop equivalents since we share logos.
func (r *HTMLRenderer) browserLogoURL(browser any) string {
	b := strings.ToLower(r.browserToString(browser))

	switch b {
	case "chrome_android":
		b = "chrome"
	case "firefox_android":
		b = "firefox"
	case "safari_ios":
		b = "safari"
	}

	return fmt.Sprintf("%s/public/img/email/%s.png", r.webStatusBaseURL, b)
}

// browserDisplayName returns a human-readable name for the browser.
func (r *HTMLRenderer) browserDisplayName(browser any) string {
	b := strings.ToLower(r.browserToString(browser))

	switch b {
	case "chrome":
		return "Chrome"
	case "chrome_android":
		return "Chrome Android"
	case "edge":
		return "Edge"
	case "firefox":
		return "Firefox"
	case "firefox_android":
		return "Firefox Android"
	case "safari":
		return "Safari"
	case "safari_ios":
		return "Safari iOS"
	}
	// Fallback for unknown
	return r.browserToString(browser)
}

func (r *HTMLRenderer) statusLogoURL(status string) string {
	return fmt.Sprintf("%s/public/img/email/%s.png", r.webStatusBaseURL, strings.ToLower(status))

}

// sortedBrowserChanges returns a list of browser changes sorted by browser name.
// This is used to ensure consistent rendering order in the template.
func (r *HTMLRenderer) sortedBrowserChanges(
	changes map[workertypes.BrowserName]*workertypes.Change[workertypes.BrowserValue]) []BrowserChangeRenderData {
	if len(changes) == 0 {
		return nil
	}

	data := make([]BrowserChangeRenderData, 0, len(changes))
	for name, change := range changes {
		if change == nil {
			continue
		}
		data = append(data, BrowserChangeRenderData{
			BrowserName: name,
			Change:      change,
			FeatureName: "",
			FeatureID:   "",
			Type:        "",
		})
	}

	// Sort by BrowserName
	slices.SortFunc(data, func(a, b BrowserChangeRenderData) int {
		if a.BrowserName < b.BrowserName {
			return -1
		} else if a.BrowserName > b.BrowserName {
			return 1
		}

		return 0
	})

	return data
}

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

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/httputils"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type SlackPayload struct {
	Text   string `json:"text,omitempty"`
	Blocks []any  `json:"blocks,omitempty"`
}

type slackSender struct {
	frontendBaseURL string
	httpClient      HTTPClient
	job             workertypes.IncomingWebhookDeliveryJob
}

func newSlackSender(frontendBaseURL string, httpClient HTTPClient,
	job workertypes.IncomingWebhookDeliveryJob) (*slackSender, error) {
	if err := httputils.ValidateSlackWebhookURL(job.WebhookURL); err != nil {
		return nil, fmt.Errorf("%w: invalid webhook URL: %w", ErrPermanentWebhook, err)
	}

	return &slackSender{
		frontendBaseURL: frontendBaseURL,
		httpClient:      httpClient,
		job:             job,
	}, nil
}

func (s *slackSender) Send(ctx context.Context) error {
	var summary workertypes.EventSummary
	if err := json.Unmarshal(s.job.SummaryRaw, &summary); err != nil {
		return fmt.Errorf("%w: failed to unmarshal summary: %w", ErrPermanentWebhook, err)
	}

	// Determine the correct results URL.
	// 1. Check if it's a feature-specific query (id:"...")
	query := s.job.Metadata.Query
	var resultsURL string
	if strings.HasPrefix(query, "id:\"") && strings.HasSuffix(query, "\"") {
		featureKey := strings.TrimSuffix(strings.TrimPrefix(query, "id:\""), "\"")
		resultsURL = fmt.Sprintf("%s/features/%s", s.frontendBaseURL, featureKey)
	} else {
		// 2. Default search results page (at the root)
		resultsURL = fmt.Sprintf("%s/?q=%s", s.frontendBaseURL, url.QueryEscape(query))
	}

	var payload SlackPayload

	if summary.SchemaVersion == "" {
		// Legacy / Simple Text Payload for backward compatibility during rollouts.
		// TODO: Remove this once all queued jobs have drained out.
		payload = SlackPayload{
			Text: fmt.Sprintf("WebStatus.dev Notification: %s\nQuery: %s\nView Results: %s",
				summary.Text, query, resultsURL),
			Blocks: nil,
		}
	} else {
		builder := &slackPayloadBuilder{
			frontendBaseURL:           s.frontendBaseURL,
			query:                     query,
			resultsURL:                resultsURL,
			summary:                   summary,
			baselineNewlyChanges:      nil,
			baselineWidelyChanges:     nil,
			baselineRegressionChanges: nil,
			allBrowserChanges:         nil,
			addedFeatures:             nil,
			removedFeatures:           nil,
			deletedFeatures:           nil,
			splitFeatures:             nil,
			movedFeatures:             nil,
			triggers:                  s.job.Triggers,
			subscriptionID:            s.job.SubscriptionID,
		}

		if err := workertypes.ParseEventSummary(s.job.SummaryRaw, builder); err != nil {
			return fmt.Errorf("%w: failed to parse event summary: %w", ErrPermanentWebhook, err)
		}

		payload = builder.buildPayload(s.job.Metadata.SearchName)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal slack payload: %w", ErrPermanentWebhook, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.job.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("%w: failed to create request: %w", ErrPermanentWebhook, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Join(ErrTransientWebhook, fmt.Errorf("network error: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	webhookErr := fmt.Errorf("webhook returned status code %d", resp.StatusCode)
	isPermanent := resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone ||
		resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden

	if !isPermanent {
		return errors.Join(ErrTransientWebhook, webhookErr)
	}

	return errors.Join(ErrPermanentWebhook, webhookErr)
}

type slackPayloadBuilder struct {
	frontendBaseURL string
	query           string
	resultsURL      string
	summary         workertypes.EventSummary

	baselineNewlyChanges      []workertypes.SummaryHighlight
	baselineWidelyChanges     []workertypes.SummaryHighlight
	baselineRegressionChanges []workertypes.SummaryHighlight
	allBrowserChanges         []browserChangeData
	addedFeatures             []workertypes.SummaryHighlight
	removedFeatures           []workertypes.SummaryHighlight
	deletedFeatures           []workertypes.SummaryHighlight
	splitFeatures             []workertypes.SummaryHighlight
	movedFeatures             []workertypes.SummaryHighlight
	triggers                  []workertypes.JobTrigger
	subscriptionID            string
}

type browserChangeData struct {
	Browser     workertypes.BrowserName
	Change      *workertypes.Change[workertypes.BrowserValue]
	FeatureName string
	FeatureID   string
	Type        workertypes.SummaryHighlightType
}

func (b *slackPayloadBuilder) VisitV1(summary workertypes.EventSummary) error {
	b.summary = summary

	filtered := workertypes.FilterHighlights(summary.Highlights, b.triggers)
	if len(filtered) != 0 {
		summary.Highlights = filtered
	}

	for _, h := range summary.Highlights {
		b.processHighlight(h)
	}

	return nil
}

func (b *slackPayloadBuilder) processHighlight(highlight workertypes.SummaryHighlight) {
	switch highlight.Type {
	case workertypes.SummaryHighlightTypeMoved:
		b.movedFeatures = append(b.movedFeatures, highlight)
	case workertypes.SummaryHighlightTypeSplit:
		b.splitFeatures = append(b.splitFeatures, highlight)
	case workertypes.SummaryHighlightTypeAdded:
		b.addedFeatures = append(b.addedFeatures, highlight)
	case workertypes.SummaryHighlightTypeRemoved:
		if highlight.BaselineChange != nil || len(highlight.BrowserChanges) > 0 {
			b.processChangedData(highlight)
		} else {
			b.removedFeatures = append(b.removedFeatures, highlight)
		}
	case workertypes.SummaryHighlightTypeDeleted:
		b.deletedFeatures = append(b.deletedFeatures, highlight)
	case workertypes.SummaryHighlightTypeChanged:
		b.processChangedData(highlight)
	}
}

func (b *slackPayloadBuilder) processChangedData(highlight workertypes.SummaryHighlight) {
	for name, change := range highlight.BrowserChanges {
		if change != nil {
			b.allBrowserChanges = append(b.allBrowserChanges, browserChangeData{
				Browser:     name,
				Change:      change,
				FeatureName: highlight.FeatureName,
				FeatureID:   highlight.FeatureID,
				Type:        highlight.Type,
			})
		}
	}

	if highlight.BaselineChange != nil {
		switch highlight.BaselineChange.To.Status {
		case workertypes.BaselineStatusNewly:
			b.baselineNewlyChanges = append(b.baselineNewlyChanges, highlight)
		case workertypes.BaselineStatusWidely:
			b.baselineWidelyChanges = append(b.baselineWidelyChanges, highlight)
		case workertypes.BaselineStatusLimited:
			b.baselineRegressionChanges = append(b.baselineRegressionChanges, highlight)
		case workertypes.BaselineStatusUnknown:
			// No-op for unknown
		default:
			// Catch any other cases if added in future
		}
	}
}

func (b *slackPayloadBuilder) buildPayload(searchName string) SlackPayload {
	var blocks []any

	title := "Update:"
	if searchName != "" {
		title = fmt.Sprintf("Weekly digest: %s", searchName)
	}
	blocks = append(blocks, headerBlock(title))

	introText := fmt.Sprintf("Here is your update for the saved search *'%s'*. \n*%s.*", searchName, b.summary.Text)
	blocks = append(blocks, sectionBlock(introText))
	blocks = append(blocks, dividerBlock())

	blocks = b.appendBaselineChanges(blocks)
	blocks = b.appendRegressions(blocks)
	blocks = b.appendBrowserChanges(blocks)
	blocks = b.appendAddedFeatures(blocks)
	blocks = b.appendSplitFeatures(blocks)
	blocks = b.appendMovedFeatures(blocks)

	unsubscribeURL := fmt.Sprintf("%s/settings/subscriptions", b.frontendBaseURL)
	if b.subscriptionID != "" {
		unsubscribeURL = fmt.Sprintf("%s?unsubscribe=%s", unsubscribeURL, b.subscriptionID)
	}
	unsubscribeText := fmt.Sprintf("You can <%s|unsubscribe> on <%s/settings/subscriptions|webstatus.dev>",
		unsubscribeURL, b.frontendBaseURL)
	blocks = append(blocks, contextBlock(map[string]any{"type": "mrkdwn", "text": unsubscribeText}))

	return SlackPayload{
		Text:   "",
		Blocks: blocks,
	}
}

func (b *slackPayloadBuilder) appendBaselineChanges(blocks []any) []any {
	if len(b.baselineNewlyChanges) > 0 {
		logoURL := fmt.Sprintf("%s/public/img/email/newly.png", b.frontendBaseURL)
		blocks = append(
			blocks,
			contextBlock(contextImageText(logoURL, "Newly Available", "*Baseline: Newly available*")...),
		)
		for _, h := range b.baselineNewlyChanges {
			featureURL := fmt.Sprintf("%s/features/%s", b.frontendBaseURL, h.FeatureID)
			txt := fmt.Sprintf(
				"<%s|%s> \n*Date:* %s",
				featureURL,
				h.FeatureName,
				formatDate(h.BaselineChange.To.LowDate),
			)
			blocks = append(blocks, sectionBlock(txt))
		}
	}

	if len(b.baselineWidelyChanges) > 0 {
		logoURL := fmt.Sprintf("%s/public/img/email/widely.png", b.frontendBaseURL)
		txt := "*Baseline: Widely available*"
		blocks = append(blocks, contextBlock(contextImageText(logoURL, "Widely Available", txt)...))
		for _, h := range b.baselineWidelyChanges {
			featureURL := fmt.Sprintf("%s/features/%s", b.frontendBaseURL, h.FeatureID)
			txt := fmt.Sprintf("<%s|%s> (<https://mdn.io|MDN>) \n*Date:* %s",
				featureURL, h.FeatureName, formatDate(h.BaselineChange.To.HighDate))
			blocks = append(blocks, sectionBlock(txt))
		}
	}

	return blocks
}

func (b *slackPayloadBuilder) appendRegressions(blocks []any) []any {
	if len(b.baselineRegressionChanges) > 0 || len(b.removedFeatures) > 0 {
		blocks = append(blocks, dividerBlock())
		blocks = append(blocks, sectionBlock("*Regressed to limited availability*"))
		if len(b.baselineRegressionChanges) > 0 {
			logoURL := fmt.Sprintf("%s/public/img/email/limited.png", b.frontendBaseURL)
			for _, h := range b.baselineRegressionChanges {
				featureURL := fmt.Sprintf("%s/features/%s", b.frontendBaseURL, h.FeatureID)
				txt := fmt.Sprintf("<%s|%s> _From Widely_", featureURL, h.FeatureName)
				blocks = append(blocks, contextBlock(contextImageText(logoURL, "Regressed", txt)...))
			}
		}
		for _, h := range b.removedFeatures {
			featureURL := fmt.Sprintf("%s/features/%s", b.frontendBaseURL, h.FeatureID)
			txt := fmt.Sprintf("<%s|%s> \n_From Newly_ \n:warning: _This feature no longer matches your saved search._",
				featureURL, h.FeatureName)
			blocks = append(blocks, sectionBlock(txt))
		}
	}

	return blocks
}

func (b *slackPayloadBuilder) appendBrowserChanges(blocks []any) []any {
	if len(b.allBrowserChanges) > 0 {
		blocks = append(blocks, dividerBlock())
		blocks = append(blocks, sectionBlock("*Browser support changed*"))
		var items []string
		for _, c := range b.allBrowserChanges {
			items = b.processBrowserChange(items, c)
		}
		if len(items) > 0 {
			blocks = append(blocks, sectionBlock(strings.Join(items, "\n")))
		}
	}

	return blocks
}

func (b *slackPayloadBuilder) processBrowserChange(items []string, c browserChangeData) []string {
	featureURL := fmt.Sprintf("%s/features/%s", b.frontendBaseURL, c.FeatureID)

	switch c.Change.To.Status {
	case workertypes.BrowserStatusAvailable:
		versionStr := ""
		if c.Change.To.Version != nil {
			versionStr = fmt.Sprintf(" in %s", *c.Change.To.Version)
		}
		txt := fmt.Sprintf("• %s: *Became available%s* (<%s|%s>)",
			formatBrowserName(c.Browser), versionStr, featureURL, c.FeatureName)
		if c.Type == workertypes.SummaryHighlightTypeRemoved {
			txt += "\n:warning: _This feature no longer matches your saved search._"
		}
		items = append(items, txt)
	case workertypes.BrowserStatusUnavailable:
		txt := fmt.Sprintf("• %s: Available in %s → *Unavailable* (<%s|%s>)",
			formatBrowserName(c.Browser), *c.Change.From.Version, featureURL, c.FeatureName)
		if c.Type == workertypes.SummaryHighlightTypeRemoved {
			txt += "\n:warning: _This feature no longer matches your saved search._"
		}
		items = append(items, txt)
	case workertypes.BrowserStatusUnknown:
		// No-op
	default:
		// Unknown cases
	}

	return items
}

func (b *slackPayloadBuilder) appendAddedFeatures(blocks []any) []any {
	if len(b.addedFeatures) > 0 {
		blocks = append(blocks, dividerBlock())
		blocks = append(blocks, sectionBlock("*Added* \n_These features now match your search criteria._"))
		items := make([]string, 0, len(b.addedFeatures))
		for _, h := range b.addedFeatures {
			items = append(items, fmt.Sprintf("• <%s/features/%s|%s>", b.frontendBaseURL, h.FeatureID, h.FeatureName))
		}
		blocks = append(blocks, sectionBlock(strings.Join(items, "\n")))
	}

	return blocks
}

func (b *slackPayloadBuilder) appendSplitFeatures(blocks []any) []any {
	if len(b.splitFeatures) > 0 {
		blocks = append(blocks, dividerBlock())
		for _, h := range b.splitFeatures {
			items := make([]string, 0, len(h.Split.To))
			for _, sub := range h.Split.To {
				noLongerStr := ""
				if sub.QueryMatch == workertypes.QueryMatchNoMatch {
					noLongerStr = " :warning: _(No longer matches)_"
				}
				items = append(
					items,
					fmt.Sprintf("• <%s/features/%s|%s>%s", b.frontendBaseURL, sub.ID, sub.Name, noLongerStr),
				)
			}
			featureURL := fmt.Sprintf("%s/features/%s", b.frontendBaseURL, h.FeatureID)
			txt := fmt.Sprintf(
				"*Split* \n<%s|%s> split into: \n%s",
				featureURL,
				h.FeatureName,
				strings.Join(items, "\n"),
			)
			blocks = append(blocks, sectionBlock(txt))
		}
	}

	return blocks
}

func (b *slackPayloadBuilder) appendMovedFeatures(blocks []any) []any {
	if len(b.movedFeatures) > 0 {
		blocks = append(blocks, dividerBlock())
		for _, h := range b.movedFeatures {
			featureURL := fmt.Sprintf("%s/features/%s", b.frontendBaseURL, h.FeatureID)
			noLongerStr := ""
			if h.Moved.To.QueryMatch == workertypes.QueryMatchNoMatch {
				noLongerStr = " :warning: _(No longer matches)_"
			}
			txt := fmt.Sprintf("*Moved/Renamed* \nRenamed from %s to <%s|%s>%s",
				h.Moved.From.Name, featureURL, h.Moved.To.Name, noLongerStr)
			blocks = append(blocks, sectionBlock(txt))
		}
	}

	return blocks
}

func headerBlock(text string) map[string]any {
	return map[string]any{
		"type": "header",
		"text": map[string]any{"type": "plain_text", "text": text, "emoji": true},
	}
}

func sectionBlock(text string) map[string]any {
	return map[string]any{
		"type": "section",
		"text": map[string]any{"type": "mrkdwn", "text": text},
	}
}

func dividerBlock() map[string]any {
	return map[string]any{"type": "divider"}
}

func contextBlock(elements ...any) map[string]any {
	return map[string]any{
		"type":     "context",
		"elements": elements,
	}
}

func contextImageText(imageURL, altText, text string) []any {
	return []any{
		map[string]any{"type": "image", "image_url": imageURL, "alt_text": altText},
		map[string]any{"type": "mrkdwn", "text": text},
	}
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.Format("2006-01-02")
}

func formatBrowserName(b workertypes.BrowserName) string {
	switch b {
	case workertypes.BrowserChrome:
		return "Chrome"
	case workertypes.BrowserChromeAndroid:
		return "Chrome Android"
	case workertypes.BrowserEdge:
		return "Edge"
	case workertypes.BrowserFirefox:
		return "Firefox"
	case workertypes.BrowserFirefoxAndroid:
		return "Firefox Android"
	case workertypes.BrowserSafari:
		return "Safari"
	case workertypes.BrowserSafariIos:
		return "Safari iOS"
	default:
		return string(b)
	}
}

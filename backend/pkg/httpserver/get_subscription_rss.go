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
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/backendtypes"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

const (
	fallbackRSSItemTitle = "WebStatus Update"
	errorRSSItemTitle    = "Error loading update"
)

// RSS struct for marshaling.
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	AtomNS  string   `xml:"xmlns:atom,attr"`
	Channel Channel  `xml:"channel"`
}

type AtomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Channel struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	AtomLinks   []AtomLink `xml:"atom:link"`
	Items       []Item     `xml:"item"`
}

type GUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr"`
}

type CDATA struct {
	Text string `xml:",cdata"`
}

type Item struct {
	Title       string `xml:"title"`
	Description CDATA  `xml:"description"`
	GUID        GUID   `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

// GetSubscriptionRSS handles the request to get an RSS feed for a subscription.
// nolint: ireturn // Signature generated from OpenAPI.
func (s *Server) GetSubscriptionRSS(
	ctx context.Context,
	request backend.GetSubscriptionRSSRequestObject,
) (backend.GetSubscriptionRSSResponseObject, error) {
	sub, err := s.wptMetricsStorer.GetSavedSearchSubscriptionPublic(ctx, request.SubscriptionId)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.GetSubscriptionRSS404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "Subscription not found",
			}, nil
		}

		return backend.GetSubscriptionRSS500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error",
		}, nil
	}

	search, err := s.wptMetricsStorer.GetSavedSearchPublic(ctx, sub.Subscribable.Id)
	if err != nil {
		if errors.Is(err, backendtypes.ErrEntityDoesNotExist) {
			return backend.GetSubscriptionRSS404JSONResponse{
				Code:    http.StatusNotFound,
				Message: "Saved search not found",
			}, nil
		}
		slog.ErrorContext(ctx, "failed to get saved search", "error", err)

		return backend.GetSubscriptionRSS500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error",
		}, nil
	}

	snapshotType := string(sub.Frequency)
	pageSize := getPageSizeOrDefault(request.Params.PageSize)
	events, nextPageToken, err := s.wptMetricsStorer.ListSavedSearchNotificationEvents(
		ctx,
		search.Id,
		snapshotType,
		pageSize,
		request.Params.PageToken,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list notification events", "error", err)

		return backend.GetSubscriptionRSS500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error",
		}, nil
	}

	channelLinkURL := s.baseURL.JoinPath("features")
	q := channelLinkURL.Query()
	q.Set("q", search.Query)
	channelLinkURL.RawQuery = q.Encode()
	channelLink := channelLinkURL.String()

	rss := RSS{
		XMLName: xml.Name{Local: "rss", Space: ""},
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: Channel{
			Title:       fmt.Sprintf("WebStatus.dev - %s", search.Name),
			Link:        channelLink,
			Description: fmt.Sprintf("RSS feed for saved search: %s", search.Name),
			Items:       make([]Item, 0, len(events)),
			AtomLinks:   nil,
		},
	}

	selfURL := s.baseURL.JoinPath("v1", "subscriptions", request.SubscriptionId, "rss")
	selfQuery := selfURL.Query()
	if request.Params.PageToken != nil {
		selfQuery.Set("page_token", *request.Params.PageToken)
	}
	if request.Params.PageSize != nil {
		selfQuery.Set("page_size", strconv.Itoa(*request.Params.PageSize))
	}
	if len(selfQuery) > 0 {
		selfURL.RawQuery = selfQuery.Encode()
	}

	rss.Channel.AtomLinks = append(rss.Channel.AtomLinks, AtomLink{
		Rel:  "self",
		Href: selfURL.String(),
	})

	if nextPageToken != nil {
		u := s.baseURL.JoinPath("v1", "subscriptions", request.SubscriptionId, "rss")
		q := u.Query()
		q.Set("page_token", *nextPageToken)
		q.Set("page_size", strconv.Itoa(pageSize))
		u.RawQuery = q.Encode()

		rss.Channel.AtomLinks = append(rss.Channel.AtomLinks, AtomLink{
			Rel:  "next",
			Href: u.String(),
		})
	}

	for _, e := range events {
		var summary workertypes.EventSummary
		var description string
		var title string
		if err := json.Unmarshal(e.Summary, &summary); err != nil {
			slog.ErrorContext(ctx, "failed to unmarshal summary", "event_id", e.ID, "error", err)

			errorHTML := fmt.Sprintf(
				"<p>Could not load details for this update. Please contact support with ID: %s</p>",
				e.ID,
			)

			rss.Channel.Items = append(rss.Channel.Items, Item{
				Title:       errorRSSItemTitle,
				Description: CDATA{Text: errorHTML},
				GUID: GUID{
					Value:       e.ID,
					IsPermaLink: "false",
				},
				PubDate: e.Timestamp.Format(time.RFC1123Z),
			})

			continue
		}

		richHTML, err := s.rssRenderer.RenderRSSDescription(summary)
		if err != nil {
			slog.ErrorContext(ctx, "failed to render RSS description", "event_id", e.ID, "error", err)
			description = summary.Text
			if description == "" {
				description = "Detailed summary unavailable"
			}
		} else {
			description = richHTML
		}
		title = summary.Text
		if title == "" {
			title = fallbackRSSItemTitle
		}

		rss.Channel.Items = append(rss.Channel.Items, Item{
			Title:       title,
			Description: CDATA{Text: description},
			GUID: GUID{
				Value:       e.ID,
				IsPermaLink: "false",
			},
			PubDate: e.Timestamp.Format(time.RFC1123Z),
		})
	}

	xmlBytes, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal RSS XML", "error", err)

		return backend.GetSubscriptionRSS500JSONResponse{
			Code:    http.StatusInternalServerError,
			Message: "Internal server error",
		}, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(xml.Header) + len(xmlBytes))
	buf.WriteString(xml.Header)
	buf.Write(xmlBytes)

	return backend.GetSubscriptionRSS200ApplicationrssXmlResponse{
		Body:          bytes.NewReader(buf.Bytes()),
		ContentLength: int64(buf.Len()),
	}, nil
}

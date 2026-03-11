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
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

type mockChannelStateManager struct {
	successCalls []successCall
	failureCalls []failureCall
	recordErr    error
}

type successCall struct {
	channelID string
	timestamp time.Time
	eventID   string
}

type failureCall struct {
	channelID   string
	err         error
	timestamp   time.Time
	isPermanent bool
	eventID     string
}

func (m *mockChannelStateManager) RecordSuccess(_ context.Context, channelID string,
	timestamp time.Time, eventID string) error {
	m.successCalls = append(m.successCalls, successCall{channelID, timestamp, eventID})

	return m.recordErr
}

func (m *mockChannelStateManager) RecordFailure(_ context.Context, channelID string,
	err error, timestamp time.Time, isPermanent bool, eventID string) error {
	m.failureCalls = append(m.failureCalls, failureCall{channelID, err, timestamp, isPermanent, eventID})

	return m.recordErr
}

func newTestIncomingWebhookDeliveryJob(url string, wType workertypes.WebhookType,
	query string, summary []byte) workertypes.IncomingWebhookDeliveryJob {
	return workertypes.IncomingWebhookDeliveryJob{
		WebhookEventID: "evt-123",
		WebhookDeliveryJob: workertypes.WebhookDeliveryJob{
			ChannelID:      "chan-1",
			WebhookURL:     url,
			WebhookType:    wType,
			SubscriptionID: "sub-456",
			Triggers:       []workertypes.JobTrigger{},
			Metadata: workertypes.DeliveryMetadata{
				EventID:     "evt-123",
				SearchID:    "search-789",
				SearchName:  "Test",
				Query:       query,
				Frequency:   workertypes.FrequencyWeekly,
				GeneratedAt: testGeneratedAt(),
			},
			SummaryRaw: summary,
		},
	}
}

func newTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode:       status,
		Status:           http.StatusText(status),
		Proto:            "HTTP/1.1",
		ProtoMajor:       1,
		ProtoMinor:       1,
		Header:           make(http.Header),
		Body:             io.NopCloser(strings.NewReader(body)),
		ContentLength:    int64(len(body)),
		TransferEncoding: []string{},
		Close:            false,
		Uncompressed:     false,
		Trailer:          make(http.Header),
		Request:          nil,
		TLS:              nil,
	}
}

func testGeneratedAt() time.Time {
	return time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
}

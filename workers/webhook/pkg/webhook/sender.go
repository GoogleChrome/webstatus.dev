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
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/workertypes"
)

type ChannelStateManager interface {
	RecordSuccess(ctx context.Context, channelID string, timestamp time.Time, eventID string) error
	RecordFailure(ctx context.Context, channelID string, err error, timestamp time.Time,
		isPermanent bool, eventID string) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Sender struct {
	httpClient      HTTPClient
	stateManager    ChannelStateManager
	frontendBaseURL string
}

func NewSender(httpClient HTTPClient, stateManager ChannelStateManager, frontendBaseURL string) *Sender {
	return &Sender{
		httpClient:      httpClient,
		stateManager:    stateManager,
		frontendBaseURL: frontendBaseURL,
	}
}

type Manager interface {
	Send(ctx context.Context) error
}

type Preparer interface {
	Prepare(job workertypes.IncomingWebhookDeliveryJob) (Manager, error)
}

func (s *Sender) SendWebhook(ctx context.Context, job workertypes.IncomingWebhookDeliveryJob) error {
	slog.InfoContext(ctx, "sending webhook", "channelID", job.ChannelID, "url", job.WebhookURL)

	var preparer Preparer
	switch job.WebhookType {
	case workertypes.WebhookTypeSlack:
		preparer = &slackPreparer{
			frontendBaseURL: s.frontendBaseURL,
			httpClient:      s.httpClient,
			stateManager:    s.stateManager,
		}
	default:
		err := fmt.Errorf("unsupported webhook type: %v", job.WebhookType)
		_ = s.stateManager.RecordFailure(ctx, job.ChannelID, err, time.Now(), true, job.WebhookEventID)

		return err
	}

	mgr, err := preparer.Prepare(job)
	if err != nil {
		return fmt.Errorf("failed to prepare webhook request: %w", err)
	}

	return mgr.Send(ctx)
}

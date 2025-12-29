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

package spanneradapters

import (
	"context"
	"time"
)

type EmailWorkerSpannerClient interface {
	RecordNotificationChannelSuccess(ctx context.Context, channelID string, timestamp time.Time, eventID string) error
	RecordNotificationChannelFailure(ctx context.Context, channelID string, errorMsg string, timestamp time.Time,
		isPermanent bool, eventID string) error
}

type EmailWorkerChannelStateManager struct {
	client EmailWorkerSpannerClient
}

func NewEmailWorkerChannelStateManager(client EmailWorkerSpannerClient) *EmailWorkerChannelStateManager {
	return &EmailWorkerChannelStateManager{client: client}
}

func (s *EmailWorkerChannelStateManager) RecordSuccess(ctx context.Context, channelID string,
	timestamp time.Time, eventID string) error {
	return s.client.RecordNotificationChannelSuccess(ctx, channelID, timestamp, eventID)
}

func (s *EmailWorkerChannelStateManager) RecordFailure(ctx context.Context, channelID string, err error,
	timestamp time.Time, permanentUserFailure bool, emailEventID string) error {
	msg := ""
	if err != nil {
		msg = err.Error()
	}

	return s.client.RecordNotificationChannelFailure(ctx, channelID, msg, timestamp, permanentUserFailure, emailEventID)
}

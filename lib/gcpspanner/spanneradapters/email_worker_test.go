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
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type mockEmailWorkerSpannerClient struct {
	successCalled bool
	successReq    struct {
		ChannelID string
		Timestamp time.Time
		EventID   string
	}
	successErr error

	failureCalled bool
	failureReq    struct {
		ChannelID   string
		Msg         string
		Timestamp   time.Time
		IsPermanent bool
		EventID     string
	}
	failureErr error
}

func (m *mockEmailWorkerSpannerClient) RecordNotificationChannelSuccess(
	_ context.Context, channelID string, timestamp time.Time, eventID string) error {
	m.successCalled = true
	m.successReq.ChannelID = channelID
	m.successReq.Timestamp = timestamp
	m.successReq.EventID = eventID

	return m.successErr
}

func (m *mockEmailWorkerSpannerClient) RecordNotificationChannelFailure(
	_ context.Context, channelID, errorMsg string, timestamp time.Time, isPermanent bool, eventID string) error {
	m.failureCalled = true
	m.failureReq.ChannelID = channelID
	m.failureReq.Msg = errorMsg
	m.failureReq.Timestamp = timestamp
	m.failureReq.IsPermanent = isPermanent
	m.failureReq.EventID = eventID

	return m.failureErr
}

func TestRecordSuccess(t *testing.T) {
	mock := new(mockEmailWorkerSpannerClient)
	adapter := NewEmailWorkerChannelStateManager(mock)

	ts := time.Now()
	eventID := "evt-1"
	err := adapter.RecordSuccess(context.Background(), "chan-1", ts, eventID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.successCalled {
		t.Error("RecordNotificationChannelSuccess not called")
	}

	expectedReq := struct {
		ChannelID string
		Timestamp time.Time
		EventID   string
	}{
		ChannelID: "chan-1",
		Timestamp: ts,
		EventID:   eventID,
	}

	if diff := cmp.Diff(expectedReq, mock.successReq); diff != "" {
		t.Errorf("RecordNotificationChannelSuccess request mismatch (-want +got):\n%s", diff)
	}
}

func TestRecordFailure(t *testing.T) {
	mock := new(mockEmailWorkerSpannerClient)
	adapter := NewEmailWorkerChannelStateManager(mock)

	testErr := errors.New("smtp error")
	ts := time.Now()
	eventID := "evt-2"
	isPermanent := true

	err := adapter.RecordFailure(context.Background(), "chan-1", testErr, ts, isPermanent, eventID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock.failureCalled {
		t.Error("RecordNotificationChannelFailure not called")
	}

	expectedReq := struct {
		ChannelID   string
		Msg         string
		Timestamp   time.Time
		IsPermanent bool
		EventID     string
	}{
		ChannelID:   "chan-1",
		Msg:         testErr.Error(),
		Timestamp:   ts,
		IsPermanent: isPermanent,
		EventID:     eventID,
	}

	if diff := cmp.Diff(expectedReq, mock.failureReq); diff != "" {
		t.Errorf("RecordNotificationChannelFailure request mismatch (-want +got):\n%s", diff)
	}
}

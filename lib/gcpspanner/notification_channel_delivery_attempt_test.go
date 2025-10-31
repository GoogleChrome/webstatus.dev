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

package gcpspanner

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

func TestNotificationChannelDeliveryAttemptOperations(t *testing.T) {
	ctx := context.Background()
	restartDatabaseContainer(t)

	// We need a channel to associate the attempt with.
	userID := uuid.NewString()
	createReq := CreateNotificationChannelRequest{
		UserID:      userID,
		Name:        "Test Channel",
		Type:        "EMAIL",
		EmailConfig: &EmailConfig{Address: "test@example.com", IsVerified: true, VerificationToken: nil},
	}
	channelID, err := spannerClient.CreateNotificationChannel(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}

	t.Run("Create", func(t *testing.T) {
		req := CreateNotificationChannelDeliveryAttemptRequest{
			ChannelID:        channelID,
			Status:           "SUCCESS",
			Details:          spanner.NullJSON{Value: map[string]interface{}{"info": "delivered"}, Valid: true},
			AttemptTimestamp: time.Now(),
		}

		attemptID, err := spannerClient.CreateNotificationChannelDeliveryAttempt(ctx, req)
		if err != nil {
			t.Fatalf("CreateNotificationChannelDeliveryAttempt failed: %v", err)
		}
		if attemptID == "" {
			t.Fatal("CreateNotificationChannelDeliveryAttempt did not return an ID")
		}

		// Verify Create (by reading directly)
		key := spanner.Key{attemptID, channelID}
		row, err := spannerClient.Single().ReadRow(ctx, notificationChannelDeliveryAttemptTable, key,
			[]string{"Status", "AttemptTimestamp"})
		if err != nil {
			t.Fatalf("failed to read back delivery attempt: %v", err)
		}

		var status string
		if err := row.ColumnByName("Status", &status); err != nil {
			t.Fatalf("failed to get status column: %v", err)
		}
		if status != "SUCCESS" {
			t.Errorf("expected status to be SUCCESS, got %s", status)
		}

		var ts time.Time
		if err := row.ColumnByName("AttemptTimestamp", &ts); err != nil {
			t.Fatalf("failed to get timestamp column: %v", err)
		}
		if ts.IsZero() {
			t.Error("expected a non-zero commit timestamp")
		}
	})
}

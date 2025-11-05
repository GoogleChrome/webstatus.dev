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
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

func TestCreateNotificationChannelDeliveryAttempt(t *testing.T) {
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
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	req := CreateNotificationChannelDeliveryAttemptRequest{
		ChannelID:        channelID,
		Status:           "SUCCESS",
		Details:          spanner.NullJSON{Value: map[string]interface{}{"info": "delivered"}, Valid: true},
		AttemptTimestamp: time.Now(),
	}

	attemptIDPtr, err := spannerClient.CreateNotificationChannelDeliveryAttempt(ctx, req)
	if err != nil {
		t.Fatalf("CreateNotificationChannelDeliveryAttempt failed: %v", err)
	}
	if attemptIDPtr == nil {
		t.Fatal("CreateNotificationChannelDeliveryAttempt did not return an ID")
	}
	attemptID := *attemptIDPtr

	// Verify Create by using the Get method.
	retrieved, err := spannerClient.GetNotificationChannelDeliveryAttempt(ctx, attemptID, channelID)
	if err != nil {
		t.Fatalf("failed to read back delivery attempt: %v", err)
	}

	if retrieved.Status != "SUCCESS" {
		t.Errorf("expected status to be SUCCESS, got %s", retrieved.Status)
	}

	if retrieved.AttemptTimestamp.IsZero() {
		t.Error("expected a non-zero commit timestamp")
	}
}

func TestCreateNotificationChannelDeliveryAttemptPruning(t *testing.T) {
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
	channelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	channelID := *channelIDPtr

	// Create more attempts than the max to trigger pruning.
	var idsToDelete []string
	var idToKeep string
	for i := 0; i < maxDeliveryAttemptsToKeep+2; i++ {
		// The sleep is a simple way to ensure distinct AttemptTimestamps for ordering.
		time.Sleep(1 * time.Millisecond)
		req := CreateNotificationChannelDeliveryAttemptRequest{
			ChannelID:        channelID,
			Status:           "SUCCESS",
			Details:          spanner.NullJSON{Value: nil, Valid: false},
			AttemptTimestamp: time.Now(),
		}
		idPtr, err := spannerClient.CreateNotificationChannelDeliveryAttempt(ctx, req)
		if err != nil {
			t.Fatalf("CreateNotificationChannelDeliveryAttempt (pruning test) failed: %v", err)
		}
		id := *idPtr
		if i < 2 { // The first two should be deleted.
			idsToDelete = append(idsToDelete, id)
		}
		if i == 5 { // This one should be kept.
			idToKeep = id
		}
	}

	// 1. Verify that the number of attempts is now capped at the max.
	// List all attempts for the channel.
	listReq := ListNotificationChannelDeliveryAttemptsRequest{
		ChannelID: channelID,
		PageSize:  10, // Arbitrary large number to get all.
		PageToken: nil,
	}
	attempts, _, err := spannerClient.ListNotificationChannelDeliveryAttempts(ctx, listReq)
	if err != nil {
		t.Fatalf("ListNotificationChannelDeliveryAttempts failed: %v", err)
	}
	if len(attempts) != maxDeliveryAttemptsToKeep {
		t.Errorf("expected %d attempts, got %d", maxDeliveryAttemptsToKeep, len(attempts))
	}
	// 2. Verify that the OLDEST attempts were the ones deleted.
	for _, id := range idsToDelete {
		_, err := spannerClient.GetNotificationChannelDeliveryAttempt(ctx, id, channelID)
		if err == nil {
			t.Errorf("expected attempt with ID %s to be deleted, but it was found", id)
		}
	}

	// 3. Verify that a NEWER attempt was kept.
	_, err = spannerClient.GetNotificationChannelDeliveryAttempt(ctx, idToKeep, channelID)
	if err != nil {
		t.Errorf("expected attempt with ID %s to be kept, but it was not found", idToKeep)
	}
}

func TestCreateNotificationChannelDeliveryAttemptConcurrency(t *testing.T) {
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
	concurrentChannelIDPtr, err := spannerClient.CreateNotificationChannel(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create notification channel for concurrency test: %v", err)
	}
	concurrentChannelID := *concurrentChannelIDPtr

	var wg sync.WaitGroup
	concurrentAttempts := 5
	attemptsPerRoutine := 3 // Total attempts = 15, which is > maxDeliveryAttemptsToKeep

	for i := 0; i < concurrentAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < attemptsPerRoutine; j++ {
				req := CreateNotificationChannelDeliveryAttemptRequest{
					ChannelID:        concurrentChannelID,
					Status:           "SUCCESS",
					Details:          spanner.NullJSON{Value: nil, Valid: false},
					AttemptTimestamp: time.Now(),
				}
				_, err := spannerClient.CreateNotificationChannelDeliveryAttempt(context.Background(), req)
				if err != nil {
					// Using t.Errorf in a goroutine is safer than t.Fatalf.
					t.Errorf("CreateNotificationChannelDeliveryAttempt in goroutine failed: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	// List all attempts for the channel.
	listReq := ListNotificationChannelDeliveryAttemptsRequest{
		ChannelID: concurrentChannelID,
		PageSize:  10, // Arbitrary large number to get all.
		PageToken: nil,
	}
	attempts, _, err := spannerClient.ListNotificationChannelDeliveryAttempts(ctx, listReq)
	if err != nil {
		t.Fatalf("ListNotificationChannelDeliveryAttempts (concurrency test) failed: %v", err)
	}
	if len(attempts) != maxDeliveryAttemptsToKeep {
		t.Errorf("expected %d attempts, got %d", maxDeliveryAttemptsToKeep, len(attempts))
	}
}

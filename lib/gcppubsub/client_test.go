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

package gcppubsub

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/GoogleChrome/webstatus.dev/lib/event"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// nolint:gochecknoglobals // WONTFIX. Used for testing.
var (
	pubsubContainer testcontainers.Container
	pubsubClient    *Client
	pubsubHost      string
)

const testProjectID = "local"

func TestMain(m *testing.M) {
	err := createPubSubContainer()
	if err != nil {
		fmt.Printf("failed to create container. error: %s", err.Error())
		os.Exit(1)
	}
	code := m.Run()
	err = terminatePubSubContainer()
	if err != nil {
		fmt.Printf("Warning: failed to terminate container. error: %s", err.Error())
		os.Exit(1)
	}
	os.Exit(code)
}

// nolint:exhaustruct // WONTFIX: external struct
func createPubSubContainer() error {
	ctx := context.Background()
	repoRoot, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		return err
	}

	goarch := runtime.GOARCH
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Dockerfile: filepath.Join(".dev", "pubsub", "Dockerfile"),
			Context:    repoRoot,
			BuildArgs:  map[string]*string{"TARGETARCH": &goarch},
			KeepImage:  true,
		},
		ExposedPorts: []string{"8060/tcp"},
		WaitingFor:   wait.ForLog("Pub/Sub setup for webstatus.dev finished"),
		Name:         "webstatus-dev-test-pubsub",
	}
	pubsubContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}

	mappedPort, err := pubsubContainer.MappedPort(ctx, "8060")
	if err != nil {
		return err
	}

	pubsubHost = fmt.Sprintf("localhost:%s", mappedPort.Port())
	err = newTestPubsubClient()
	if err != nil {
		return err
	}

	return nil
}

func newTestPubsubClient() error {
	var err error
	// Set this for the sdk to automatically detect.
	os.Setenv("PUBSUB_EMULATOR_HOST", pubsubHost)
	pubsubClient, err = NewClient(context.Background(), testProjectID)
	if err != nil {
		if unsetErr := os.Unsetenv("PUBSUB_EMULATOR_HOST"); unsetErr != nil {
			return fmt.Errorf("failed to unset env. %s", unsetErr.Error())
		}
		pubsubClient.Close()
		if terminateErr := pubsubContainer.Terminate(context.Background()); terminateErr != nil {
			return fmt.Errorf("failed to terminate container. %s", terminateErr.Error())
		}

		return fmt.Errorf("failed to create client. %s", err.Error())
	}

	return nil
}

func terminatePubSubContainer() error {
	if unsetErr := os.Unsetenv("PUBSUB_EMULATOR_HOST"); unsetErr != nil {
		return fmt.Errorf("failed to unset env. %s", unsetErr.Error())
	}
	pubsubClient.Close()
	if err := pubsubContainer.Terminate(context.Background()); err != nil {
		return fmt.Errorf("failed to terminate datastore. %s", err.Error())
	}

	return nil
}

func createTestTopic(t *testing.T, topicID string) {
	ctx := context.Background()
	// nolint:exhaustruct // WONTFIX: external struct
	topicpb := &pubsubpb.Topic{
		Name: fmt.Sprintf("projects/%s/topics/%s", testProjectID, topicID),
	}
	_, err := pubsubClient.client.TopicAdminClient.CreateTopic(ctx, topicpb)
	if err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}
}

func createTestSubscription(t *testing.T, topicID, subID string, seconds int32) {
	ctx := context.Background()
	topicName := fmt.Sprintf("projects/%s/topics/%s", testProjectID, topicID)
	// nolint:exhaustruct // WONTFIX: external struct
	subpb := &pubsubpb.Subscription{
		Name:               fmt.Sprintf("projects/%s/subscriptions/%s", testProjectID, subID),
		Topic:              topicName,
		AckDeadlineSeconds: seconds,
	}
	_, err := pubsubClient.client.SubscriptionAdminClient.CreateSubscription(ctx, subpb)
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
}

func TestPublishAndSubscribe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topicID := "test-topic"
	subID := "test-sub"

	// 1. Create Topic
	createTestTopic(t, topicID)

	// 2. Create Subscription
	createTestSubscription(t, topicID, subID, 10)

	// 3. Publish Message
	msgData := []byte("hello-world")
	msgID, err := pubsubClient.Publish(ctx, topicID, msgData)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if msgID == "" {
		t.Error("Expected non-empty message ID")
	}

	// 4. Subscribe and Verify
	received := make(chan string, 1)

	// Start subscriber in goroutine
	go func() {
		err := pubsubClient.Subscribe(ctx, subID, func(_ context.Context, data []byte) error {
			received <- string(data)

			return nil // ACK
		})
		if err != nil && ctx.Err() == nil {
			t.Errorf("Subscribe failed: %v", err)
		}
	}()

	select {
	case data := <-received:
		if data != string(msgData) {
			t.Errorf("Expected message %q, got %q", string(msgData), data)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for message")
	}
}

func TestSubscribe_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	topicID := "error-handling-topic"
	subID := "error-handling-sub"

	// Setup Topic & Sub
	createTestTopic(t, topicID)
	createTestSubscription(t, topicID, subID, 1) // Short deadline for NACK retry test

	// Case 1: Transient Error (Should Retry)
	t.Run("TransientError_Retries", func(t *testing.T) {
		_, err := pubsubClient.Publish(ctx, topicID, []byte("retry-me"))
		if err != nil {
			// Should not error out of Publish
			t.Fatalf("Publish failed: %v", err)
		}

		var attempts atomic.Int32
		done := make(chan struct{})

		go func() {
			err := pubsubClient.Subscribe(ctx, subID, func(_ context.Context, data []byte) error {
				if string(data) != "retry-me" {
					return nil // Ignore other messages
				}

				count := attempts.Add(1)
				if count == 1 {
					// First attempt: Simulate Transient Error
					return event.ErrTransientFailure
				}
				// Second attempt: Success
				close(done)

				return nil
			})
			if err != nil {
				// Should not error out of Subscribe
				t.Errorf("Subscribe failed: %v", err)
			}
		}()

		select {
		case <-done:
			if attempts.Load() < 2 {
				t.Errorf("Expected retries, but succeeded on attempt %d", attempts.Load())
			}
		case <-ctx.Done():
			t.Fatal("Timeout waiting for retry")
		}
	})

	// Case 2: Permanent Error (Should ACK and NOT Retry)
	t.Run("PermanentError_NoRetry", func(t *testing.T) {
		// New subscription to isolate logic
		permSubID := "perm-error-sub"
		createTestSubscription(t, topicID, permSubID, 1)

		_, err := pubsubClient.Publish(ctx, topicID, []byte("bad-data"))
		if err != nil {
			// Should not error out of Publish
			t.Fatalf("Publish failed: %v", err)
		}

		var attempts atomic.Int32

		go func() {
			err := pubsubClient.Subscribe(ctx, permSubID, func(_ context.Context, data []byte) error {
				if string(data) != "bad-data" {
					return nil
				}
				attempts.Add(1)
				// Simulate Permanent Error (should ACK)
				return errors.New("some permanent error")
			})
			if err != nil {
				// Should not error out of Subscribe
				t.Errorf("Subscribe failed: %v", err)
			}
		}()

		// Wait a bit to ensure it DOESN'T retry
		time.Sleep(3 * time.Second)

		// If it retried, attempts would be > 1 because AckDeadline is 1s
		if count := attempts.Load(); count > 1 {
			t.Errorf("Expected 1 attempt (ACK), got %d (NACK/Retry?)", count)
		}
	})
}

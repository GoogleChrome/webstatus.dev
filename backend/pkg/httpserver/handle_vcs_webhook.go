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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

const maxWebhookBodyBytes = 10 << 20 // 10MB payload ceiling

// HandleVCSWebhook handles POST /v1/vcs/webhooks/{provider}.
//
//nolint:ireturn, revive // Expected ireturn for openapi generation.
func (s *Server) HandleVCSWebhook(
	ctx context.Context,
	request backend.HandleVCSWebhookRequestObject,
) (backend.HandleVCSWebhookResponseObject, error) {
	switch request.Provider {
	case vcsProviderGitHub:
		return s.handleGitHubWebhook(ctx, request)
	default:
		return backend.HandleVCSWebhook400JSONResponse(backend.BasicErrorModel{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("unsupported VCS provider: %s", request.Provider),
		}), nil
	}
}

// =========================================================================
// GitHub Webhook Handling
// =========================================================================

type gitHubPushPayload struct {
	Ref          string            `json:"ref"`
	Before       string            `json:"before"`
	After        string            `json:"after"`
	Repository   gitHubRepoPayload `json:"repository"`
	Installation gitHubInstPayload `json:"installation"`
}

type gitHubRepoPayload struct {
	ID            int64              `json:"id"`
	Name          string             `json:"name"`
	FullName      string             `json:"full_name"`
	DefaultBranch string             `json:"default_branch"`
	Owner         gitHubOwnerPayload `json:"owner"`
}

type gitHubOwnerPayload struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type gitHubInstPayload struct {
	ID int64 `json:"id"`
}

func (s *Server) isInvalidSignature(ctx context.Context, rawBody []byte, sigHeader *string) bool {
	var sig string
	if sigHeader != nil {
		sig = *sigHeader
	}
	if err := s.webhookVerifier.VerifySignature(rawBody, sig); err != nil {
		slog.WarnContext(ctx, "invalid webhook signature", "error", err, "header", sig)

		return true
	}

	return false
}

//nolint:ireturn, revive, gocognit // Expected ireturn for openapi generation.
func (s *Server) handleGitHubWebhook(
	ctx context.Context,
	request backend.HandleVCSWebhookRequestObject,
) (backend.HandleVCSWebhookResponseObject, error) {
	if request.Params.XGitHubDelivery == nil || *request.Params.XGitHubDelivery == "" {
		return backend.HandleVCSWebhook400JSONResponse(backend.BasicErrorModel{
			Code:    http.StatusBadRequest,
			Message: "missing X-GitHub-Delivery header",
		}), nil
	}

	var rawBody []byte
	if request.Body != nil {
		var err error
		rawBody, err = io.ReadAll(io.LimitReader(request.Body, maxWebhookBodyBytes))
		if err != nil {
			slog.WarnContext(ctx, "failed to read webhook payload", "error", err)

			return backend.HandleVCSWebhook400JSONResponse(backend.BasicErrorModel{
				Code:    http.StatusBadRequest,
				Message: "failed to read webhook payload",
			}), nil
		}
	}

	if s.isInvalidSignature(ctx, rawBody, request.Params.XHubSignature256) {
		return backend.HandleVCSWebhook401JSONResponse(backend.BasicErrorModel{
			Code:    http.StatusUnauthorized,
			Message: "invalid webhook signature",
		}), nil
	}

	deliveryGUID := *request.Params.XGitHubDelivery
	eventType := getGitHubEventTypeOrDefault(request.Params.XGitHubEvent)
	now := time.Now().UTC()

	var payload gitHubPushPayload
	if unmarshalErr := json.Unmarshal(rawBody, &payload); unmarshalErr != nil {
		slog.WarnContext(ctx, "failed to unmarshal webhook payload", "error", unmarshalErr)
	}
	repoID := strconv.FormatInt(payload.Repository.ID, 10)

	delivery := gcpspanner.VCSWebhookDelivery{
		VCSProvider:     gcpspanner.VCSProviderGitHub,
		DeliveryGUID:    deliveryGUID,
		EventType:       eventType,
		VCSRepositoryID: repoID,
		ReceivedAt:      now,
	}

	// Dispatch the scan task to Pub/Sub BEFORE recording the delivery in Spanner.
	// This ensures that if publishing encounters a transient error, the handler
	// returns HTTP 500 without writing to Spanner, allowing GitHub to retry the
	// webhook delivery without being prematurely ignored as a duplicate.
	// Downstream scanner synchronization is fully idempotent.
	if eventType == "push" && s.eventPublisher != nil {
		if dispatchErr := s.dispatchGitHubPushScanTask(ctx, rawBody); dispatchErr != nil {
			//nolint:nilerr // Return 500 response per OpenAPI spec
			return backend.HandleVCSWebhook500JSONResponse(backend.BasicErrorModel{
				Code:    http.StatusInternalServerError,
				Message: "failed to publish scan task",
			}), nil
		}
	}

	if s.wptMetricsStorer != nil {
		isNew, recordErr := s.wptMetricsStorer.RecordVCSWebhookDelivery(ctx, delivery)
		if recordErr != nil {
			slog.ErrorContext(ctx, "failed to record webhook delivery", "error", recordErr, "guid", deliveryGUID)

			return backend.HandleVCSWebhook500JSONResponse(backend.BasicErrorModel{
				Code:    http.StatusInternalServerError,
				Message: "failed to record webhook delivery",
			}), nil
		}
		if !isNew {
			slog.InfoContext(ctx, "duplicate webhook delivery recorded", "guid", deliveryGUID)
		}
	}

	return backend.HandleVCSWebhook202Response{}, nil
}

func (s *Server) dispatchGitHubPushScanTask(
	ctx context.Context,
	rawBody []byte,
) error {
	var payload gitHubPushPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		slog.WarnContext(ctx, "failed to unmarshal push payload", "error", err)

		return fmt.Errorf("failed to unmarshal push payload: %w", err)
	}

	isDefault := payload.Ref == fmt.Sprintf("refs/heads/%s", payload.Repository.DefaultBranch)

	task := codescantaskv1.CodeScanTaskEvent{
		VCSProvider:        vcsProviderGitHub,
		VCSInstallationID:  strconv.FormatInt(payload.Installation.ID, 10),
		VCSRepositoryID:    strconv.FormatInt(payload.Repository.ID, 10),
		RepositoryFullName: payload.Repository.FullName,
		CommitSHA:          payload.After,
		Branch:             payload.Ref,
		IsDefaultBranch:    isDefault,
		ModifiedFiles:      nil,
	}

	if pubErr := s.eventPublisher.PublishCodeScanTask(ctx, task); pubErr != nil {
		slog.ErrorContext(ctx, "failed to publish scan task", "error", pubErr, "repo", task.RepositoryFullName)

		return fmt.Errorf("failed to publish scan task: %w", pubErr)
	}

	return nil
}

func getGitHubEventTypeOrDefault(headerVal *string) string {
	if headerVal != nil && *headerVal != "" {
		return *headerVal
	}

	return "push"
}

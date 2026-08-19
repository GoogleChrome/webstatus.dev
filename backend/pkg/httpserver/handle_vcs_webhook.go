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
	"log/slog"
	"net/http"
	"strconv"
	"time"

	codescantaskv1 "github.com/GoogleChrome/webstatus.dev/lib/event/codescantask/v1"
	"github.com/GoogleChrome/webstatus.dev/lib/gcpspanner"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

type pushPayload struct {
	Ref          string `json:"ref"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Repository   repo   `json:"repository"`
	Installation inst   `json:"installation"`
}

type repo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	Owner         owner  `json:"owner"`
}

type owner struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type inst struct {
	ID int64 `json:"id"`
}

func (s *Server) isInvalidSignature(ctx context.Context, rawBody []byte, sigHeader *string) bool {
	if s.webhookVerifier == nil {
		return false
	}
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

func (s *Server) dispatchPushScanTask(
	ctx context.Context,
	provider string,
	rawBody []byte,
) error {
	var payload pushPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		slog.WarnContext(ctx, "failed to unmarshal push payload", "error", err)

		return fmt.Errorf("failed to unmarshal push payload: %w", err)
	}

	isDefault := payload.Ref == fmt.Sprintf("refs/heads/%s", payload.Repository.DefaultBranch)

	task := codescantaskv1.CodeScanTaskEvent{
		VCSProvider:        provider,
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

func getEventTypeOrDefault(headerVal *string) string {
	if headerVal != nil && *headerVal != "" {
		return *headerVal
	}

	return "push"
}

//nolint:ireturn, revive, gocognit // Expected ireturn for openapi generation.
func (s *Server) HandleVCSWebhook(
	ctx context.Context,
	request backend.HandleVCSWebhookRequestObject,
) (backend.HandleVCSWebhookResponseObject, error) {
	if request.Provider != "github" {
		return backend.HandleVCSWebhook400JSONResponse(backend.BasicErrorModel{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("unsupported VCS provider: %s", request.Provider),
		}), nil
	}

	if request.Params.XGitHubDelivery == nil || *request.Params.XGitHubDelivery == "" {
		return backend.HandleVCSWebhook400JSONResponse(backend.BasicErrorModel{
			Code:    http.StatusBadRequest,
			Message: "missing X-GitHub-Delivery header",
		}), nil
	}

	rawBody, err := json.Marshal(request.Body)
	if err != nil {
		slog.WarnContext(ctx, "failed to marshal webhook payload", "error", err)

		return backend.HandleVCSWebhook400JSONResponse(backend.BasicErrorModel{ //nolint:nilerr // Return 400 for bad JSON
			Code:    http.StatusBadRequest,
			Message: "invalid webhook payload json",
		}), nil
	}

	if s.isInvalidSignature(ctx, rawBody, request.Params.XHubSignature256) {
		return backend.HandleVCSWebhook401JSONResponse(backend.BasicErrorModel{
			Code:    http.StatusUnauthorized,
			Message: "invalid webhook signature",
		}), nil
	}

	deliveryGUID := *request.Params.XGitHubDelivery
	eventType := getEventTypeOrDefault(request.Params.XGitHubEvent)
	now := time.Now().UTC()

	var payload pushPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		slog.WarnContext(ctx, "failed to unmarshal webhook payload", "error", err)
	}
	repoID := strconv.FormatInt(payload.Repository.ID, 10)

	delivery := gcpspanner.VCSWebhookDelivery{
		VCSProvider:     gcpspanner.VCSProviderGitHub,
		DeliveryGUID:    deliveryGUID,
		EventType:       eventType,
		VCSRepositoryID: repoID,
		ReceivedAt:      now,
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
			slog.InfoContext(ctx, "duplicate webhook delivery ignored", "guid", deliveryGUID)

			return backend.HandleVCSWebhook202Response{}, nil
		}
	}

	if eventType == "push" && s.eventPublisher != nil {
		if err := s.dispatchPushScanTask(ctx, request.Provider, rawBody); err != nil {
			//nolint:nilerr // Return 500 response per OpenAPI spec
			return backend.HandleVCSWebhook500JSONResponse(backend.BasicErrorModel{
				Code:    http.StatusInternalServerError,
				Message: "failed to publish scan task",
			}), nil
		}
	}

	return backend.HandleVCSWebhook202Response{}, nil
}

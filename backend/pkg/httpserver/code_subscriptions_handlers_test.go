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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

type customMockCodeSubscriptionStorer struct {
	*MockWPTMetricsStorer
	listSubsOutput []backend.CodeSubscriptionResponse
	listSubsErr    error
	deleteErr      error
}

func newMockCodeSubscriptionStorer(
	t *testing.T,
	listOutput []backend.CodeSubscriptionResponse,
	listErr error,
	deleteErr error,
) *customMockCodeSubscriptionStorer {
	t.Helper()

	return &customMockCodeSubscriptionStorer{
		MockWPTMetricsStorer: newDefaultMockWPTMetricsStorer(t),
		listSubsOutput:       listOutput,
		listSubsErr:          listErr,
		deleteErr:            deleteErr,
	}
}

func (m *customMockCodeSubscriptionStorer) ListCodeSubscriptions(
	_ context.Context,
	_, _ string,
) ([]backend.CodeSubscriptionResponse, error) {
	return m.listSubsOutput, m.listSubsErr
}

func (m *customMockCodeSubscriptionStorer) DeleteCodeSubscription(
	_ context.Context,
	_ string,
) error {
	return m.deleteErr
}

func TestListCodeSubscriptions(t *testing.T) {
	testUser := &auth.User{ID: "user-123", GitHubUserID: new("12345")}
	now := time.Now().UTC()

	sampleSub := backend.CodeSubscriptionResponse{
		Id:                 "sub-1",
		VcsProvider:        "github",
		VcsInstallationId:  new("inst-1"),
		VcsRepositoryId:    "repo-1",
		RepositoryOwner:    "GoogleChrome",
		RepositoryName:     "webstatus.dev",
		RepositoryFullName: "GoogleChrome/webstatus.dev",
		FeatureId:          new("view-transitions"),
		TargetQuery:        "id:view-transitions",
		Triggers:           []backend.SubscriptionTriggerWritable{backend.SubscriptionTriggerFeatureBaselineToWidely},
		RawDirective:       new("// @webstatus: id:view-transitions"),
		Occurrences: []backend.SubscriptionOccurrence{
			{FilePath: "src/app.ts", LineNumber: 10, CommentSnippet: "// @webstatus: id:view-transitions"},
		},
		OccurrenceCount: 1,
		Status:          backend.ACTIVE,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	testCases := []struct {
		name         string
		userInCtx    *auth.User
		storerOutput []backend.CodeSubscriptionResponse
		storerErr    error
		expectedCode int
	}{
		{
			name:         "Success",
			userInCtx:    testUser,
			storerOutput: []backend.CodeSubscriptionResponse{sampleSub},
			storerErr:    nil,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Missing User in Context",
			userInCtx:    nil,
			storerOutput: nil,
			storerErr:    nil,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "Database Error",
			userInCtx:    testUser,
			storerOutput: nil,
			storerErr:    errors.New("spanner failure"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storer := newMockCodeSubscriptionStorer(t, tc.storerOutput, tc.storerErr, nil)
			server := setupTestServer(t, withCustomStorer(storer))

			ctx := context.Background()
			if tc.userInCtx != nil {
				ctx = httpmiddlewares.AuthenticatedUserToContext(ctx, tc.userInCtx)
			}

			req := backend.ListCodeSubscriptionsRequestObject{
				Provider:     "github",
				RepositoryId: "repo-1",
			}
			resp, err := server.ListCodeSubscriptions(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			rec := httptest.NewRecorder()
			if err := resp.VisitListCodeSubscriptionsResponse(rec); err != nil {
				t.Fatalf("failed to visit response: %v", err)
			}

			if rec.Code != tc.expectedCode {
				t.Errorf("expected status code %d, got %d", tc.expectedCode, rec.Code)
			}
		})
	}
}

func TestDeleteCodeSubscription_TableDriven(t *testing.T) {
	testUser := &auth.User{ID: "user-123", GitHubUserID: new("12345")}

	testCases := []struct {
		name         string
		userInCtx    *auth.User
		deleteErr    error
		expectedCode int
	}{
		{
			name:         "Success",
			userInCtx:    testUser,
			deleteErr:    nil,
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "Missing User in Context",
			userInCtx:    nil,
			deleteErr:    nil,
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "Database Error",
			userInCtx:    testUser,
			deleteErr:    errors.New("spanner failure"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storer := newMockCodeSubscriptionStorer(t, nil, nil, tc.deleteErr)
			server := setupTestServer(t, withCustomStorer(storer))

			ctx := context.Background()
			if tc.userInCtx != nil {
				ctx = httpmiddlewares.AuthenticatedUserToContext(ctx, tc.userInCtx)
			}

			req := backend.DeleteCodeSubscriptionRequestObject{
				Id: "sub-1",
			}
			resp, err := server.DeleteCodeSubscription(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			rec := httptest.NewRecorder()
			if err := resp.VisitDeleteCodeSubscriptionResponse(rec); err != nil {
				t.Fatalf("failed to visit response: %v", err)
			}

			if rec.Code != tc.expectedCode {
				t.Errorf("expected status code %d, got %d", tc.expectedCode, rec.Code)
			}
		})
	}
}

func TestVCSInstallationsAndRepositories_Handlers(t *testing.T) {
	testUser := &auth.User{ID: "user-123", GitHubUserID: new("12345")}
	server := setupTestServer(t)

	ctx := httpmiddlewares.AuthenticatedUserToContext(context.Background(), testUser)

	instResp, err := server.ListVCSInstallations(ctx, backend.ListVCSInstallationsRequestObject{Provider: "github"})
	if err != nil {
		t.Fatalf("ListVCSInstallations error: %v", err)
	}
	instRec := httptest.NewRecorder()
	if err := instResp.VisitListVCSInstallationsResponse(instRec); err != nil {
		t.Fatalf("failed to visit installations response: %v", err)
	}
	if instRec.Code != http.StatusOK {
		t.Errorf("expected 200 response, got %d", instRec.Code)
	}

	repoResp, err := server.ListVCSRepositories(ctx, backend.ListVCSRepositoriesRequestObject{Provider: "github"})
	if err != nil {
		t.Fatalf("ListVCSRepositories error: %v", err)
	}
	repoRec := httptest.NewRecorder()
	if err := repoResp.VisitListVCSRepositoriesResponse(repoRec); err != nil {
		t.Fatalf("failed to visit repositories response: %v", err)
	}
	if repoRec.Code != http.StatusOK {
		t.Errorf("expected 200 response, got %d", repoRec.Code)
	}

	webhookBody := backend.HandleVCSWebhookJSONRequestBody{}
	webhookReq := backend.HandleVCSWebhookRequestObject{
		Provider: "github",
		Params: backend.HandleVCSWebhookParams{
			XHubSignature256: nil,
			XGitHubDelivery:  nil,
			XGitHubEvent:     nil,
		},
		Body: &webhookBody,
	}
	webhookResp, err := server.HandleVCSWebhook(context.Background(), webhookReq)
	if err != nil {
		t.Fatalf("HandleVCSWebhook error: %v", err)
	}
	webhookRec := httptest.NewRecorder()
	if err := webhookResp.VisitHandleVCSWebhookResponse(webhookRec); err != nil {
		t.Fatalf("failed to visit webhook response: %v", err)
	}
	if webhookRec.Code != http.StatusAccepted {
		t.Errorf("expected 202 response, got %d", webhookRec.Code)
	}
}

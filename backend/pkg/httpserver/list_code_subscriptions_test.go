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
		name                     string
		userInCtx                *auth.User
		listCodeSubscriptionsCfg *MockListCodeSubscriptionsConfig
		expectedCode             int
	}{
		{
			name:      "Success",
			userInCtx: testUser,
			listCodeSubscriptionsCfg: &MockListCodeSubscriptionsConfig{
				expectedProvider:     "github",
				expectedRepositoryID: "repo-1",
				output:               []backend.CodeSubscriptionResponse{sampleSub},
				err:                  nil,
			},
			expectedCode: http.StatusOK,
		},
		{
			name:                     "Missing User in Context",
			userInCtx:                nil,
			listCodeSubscriptionsCfg: nil,
			expectedCode:             http.StatusInternalServerError,
		},
		{
			name:      "Database Error",
			userInCtx: testUser,
			listCodeSubscriptionsCfg: &MockListCodeSubscriptionsConfig{
				expectedProvider:     "github",
				expectedRepositoryID: "repo-1",
				output:               nil,
				err:                  errors.New("spanner failure"),
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storer := newDefaultMockWPTMetricsStorer(t)
			storer.listCodeSubscriptionsCfg = tc.listCodeSubscriptionsCfg
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

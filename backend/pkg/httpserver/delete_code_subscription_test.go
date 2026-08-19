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

	"github.com/GoogleChrome/webstatus.dev/lib/auth"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/GoogleChrome/webstatus.dev/lib/httpmiddlewares"
)

func TestDeleteCodeSubscription(t *testing.T) {
	testUser := &auth.User{ID: "user-123", GitHubUserID: new("12345")}

	testCases := []struct {
		name                      string
		userInCtx                 *auth.User
		deleteCodeSubscriptionCfg *MockDeleteCodeSubscriptionConfig
		expectedCode              int
	}{
		{
			name:      "Success",
			userInCtx: testUser,
			deleteCodeSubscriptionCfg: &MockDeleteCodeSubscriptionConfig{
				expectedSubscriptionID: "sub-1",
				err:                    nil,
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name:                      "Missing User in Context",
			userInCtx:                 nil,
			deleteCodeSubscriptionCfg: nil,
			expectedCode:              http.StatusInternalServerError,
		},
		{
			name:      "Database Error",
			userInCtx: testUser,
			deleteCodeSubscriptionCfg: &MockDeleteCodeSubscriptionConfig{
				expectedSubscriptionID: "sub-1",
				err:                    errors.New("spanner failure"),
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storer := newDefaultMockWPTMetricsStorer(t)
			storer.deleteCodeSubscriptionCfg = tc.deleteCodeSubscriptionCfg
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

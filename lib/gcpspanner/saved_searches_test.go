// Copyright 2024 Google LLC
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
	"errors"
	"testing"
)

func TestCreateNewUserSavedSearch(t *testing.T) {
	restartDatabaseContainer(t)
	ctx := context.Background()

	savedSearchesConfig := SearchConfig{
		MaxOwnedSearchesPerUser: 2,
	}

	savedSearchID1, err := spannerClient.CreateNewUserSavedSearch(ctx, savedSearchesConfig, NewSavedSearchRequest{
		Name:        "my little search",
		Query:       "group:css",
		OwnerUserID: "userID1",
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if savedSearchID1 == nil {
		t.Error("expected non-nil id.")
	}

	savedSearchID2, err := spannerClient.CreateNewUserSavedSearch(ctx, savedSearchesConfig, NewSavedSearchRequest{
		Name:        "my little search part 2",
		Query:       "group:avif",
		OwnerUserID: "userID1",
	})
	if err != nil {
		t.Errorf("expected nil error. received %s", err)
	}
	if savedSearchID2 == nil {
		t.Error("expected non-nil id.")
	}

	savedSearchID3, err := spannerClient.CreateNewUserSavedSearch(ctx, savedSearchesConfig, NewSavedSearchRequest{
		Name:        "my little search part 3",
		Query:       "name:subgrid",
		OwnerUserID: "userID1",
	})
	if !errors.Is(err, ErrOwnerSavedSearchLimitExceeded) {
		t.Errorf("unexpected error. received %v", err)
	}
	if savedSearchID3 != nil {
		t.Error("expected nil id.")
	}
}

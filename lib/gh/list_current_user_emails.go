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

package gh

import (
	"context"

	"github.com/google/go-github/v75/github"
)

type UserEmail struct {
	Email    string
	Verified bool
}

func (c *UserGitHubClient) ListEmails(ctx context.Context) ([]*UserEmail, error) {
	var emails []*UserEmail
	p := newPaginator(c.usersClient.ListEmails, func(item *github.UserEmail) (*UserEmail, bool) {
		if item == nil || item.Email == nil {
			return nil, false
		}
		var verified bool
		if item.Verified != nil {
			verified = *item.Verified
		}

		return &UserEmail{
			Email:    *item.Email,
			Verified: verified,
		}, true
	})
	for p.HasNextPage() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		emails = append(emails, page...)
	}

	return emails, nil
}

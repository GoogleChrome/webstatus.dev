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

package gcpspanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const vcsInstallationTable = "VCSInstallations"

var (
	// ErrVCSInstallationNotFound indicates that the installation record does not exist.
	ErrVCSInstallationNotFound = errors.New("vcs installation not found")
)

// VCSProvider represents a supported version control system provider.
type VCSProvider string

const (
	VCSProviderGitHub VCSProvider = "github"
)

// GitHubPermissionLevel represents a GitHub App permission level.
type GitHubPermissionLevel string

const (
	GitHubPermissionLevelRead  GitHubPermissionLevel = "read"
	GitHubPermissionLevelWrite GitHubPermissionLevel = "write"
	GitHubPermissionLevelAdmin GitHubPermissionLevel = "admin"
)

// GitHubPermissions represents GitHub App permission levels for an installation.
type GitHubPermissions struct {
	Issues       *GitHubPermissionLevel `json:"issues,omitempty"`
	Contents     *GitHubPermissionLevel `json:"contents,omitempty"`
	Metadata     *GitHubPermissionLevel `json:"metadata,omitempty"`
	PullRequests *GitHubPermissionLevel `json:"pull_requests,omitempty"`
	Workflows    *GitHubPermissionLevel `json:"workflows,omitempty"`
	Actions      *GitHubPermissionLevel `json:"actions,omitempty"`
}

// VCSPermissions encapsulates provider-specific permissions.
type VCSPermissions struct {
	GitHub *GitHubPermissions `json:"github,omitempty"`
}

// VCSInstallation represents an installed GitHub App instance for an account.
type VCSInstallation struct {
	ID                  string
	VCSProvider         VCSProvider
	VCSInstallationID   string
	AccountLogin        string
	AccountType         string
	RepositorySelection string
	Permissions         VCSPermissions
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type spannerVCSInstallation struct {
	ID                  string           `spanner:"ID"`
	VCSProvider         string           `spanner:"VCSProvider"`
	VCSInstallationID   string           `spanner:"VCSInstallationID"`
	AccountLogin        string           `spanner:"AccountLogin"`
	AccountType         string           `spanner:"AccountType"`
	RepositorySelection string           `spanner:"RepositorySelection"`
	Permissions         spanner.NullJSON `spanner:"Permissions"`
	CreatedAt           time.Time        `spanner:"CreatedAt"`
	UpdatedAt           time.Time        `spanner:"UpdatedAt"`
}

type vcsInstallationMapper struct{}

func (m vcsInstallationMapper) Table() string { return vcsInstallationTable }

func (m vcsInstallationMapper) SelectOne(id string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, AccountLogin, AccountType,
			RepositorySelection, Permissions, CreatedAt, UpdatedAt
		FROM VCSInstallations
		WHERE ID = @id`,
		Params: map[string]any{"id": id},
	}
}

func (m vcsInstallationMapper) GetKeyFromExternal(in VCSInstallation) string {
	return in.ID
}

func extractPermissionsValue(in VCSInstallation) any {
	switch in.VCSProvider {
	case VCSProviderGitHub:
		if in.Permissions.GitHub != nil {
			return in.Permissions.GitHub
		}
	}

	return map[string]any{}
}

func (m vcsInstallationMapper) NewEntity(id string, in VCSInstallation) (spannerVCSInstallation, error) {
	entityID := id
	if in.ID != "" {
		entityID = in.ID
	}

	return spannerVCSInstallation{
		ID:                  entityID,
		VCSProvider:         string(in.VCSProvider),
		VCSInstallationID:   in.VCSInstallationID,
		AccountLogin:        in.AccountLogin,
		AccountType:         in.AccountType,
		RepositorySelection: in.RepositorySelection,
		Permissions: spanner.NullJSON{
			Value: extractPermissionsValue(in),
			Valid: true,
		},
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}, nil
}

func (m vcsInstallationMapper) Merge(
	in VCSInstallation,
	existing spannerVCSInstallation,
) spannerVCSInstallation {
	return spannerVCSInstallation{
		ID:                  existing.ID,
		VCSProvider:         string(in.VCSProvider),
		VCSInstallationID:   in.VCSInstallationID,
		AccountLogin:        in.AccountLogin,
		AccountType:         in.AccountType,
		RepositorySelection: in.RepositorySelection,
		Permissions: spanner.NullJSON{
			Value: extractPermissionsValue(in),
			Valid: true,
		},
		CreatedAt: existing.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}

func loadVCSPermissions(provider VCSProvider, permissionsJSON spanner.NullJSON) (VCSPermissions, error) {
	var ret VCSPermissions
	if !permissionsJSON.Valid || permissionsJSON.Value == nil {
		return ret, nil
	}

	bytes, err := json.Marshal(permissionsJSON.Value)
	if err != nil {
		return ret, fmt.Errorf("failed to marshal permissions json: %w", err)
	}

	switch provider {
	case VCSProviderGitHub:
		var gh GitHubPermissions
		if err := json.Unmarshal(bytes, &gh); err != nil {
			return ret, fmt.Errorf("failed to unmarshal github permissions: %w", err)
		}
		if gh.Issues != nil || gh.Contents != nil || gh.Metadata != nil ||
			gh.PullRequests != nil || gh.Workflows != nil || gh.Actions != nil {
			ret.GitHub = &gh
		}
	}

	return ret, nil
}

func (s *spannerVCSInstallation) toVCSInstallation() (*VCSInstallation, error) {
	provider := VCSProvider(s.VCSProvider)
	permissions, err := loadVCSPermissions(provider, s.Permissions)
	if err != nil {
		return nil, err
	}

	return &VCSInstallation{
		ID:                  s.ID,
		VCSProvider:         provider,
		VCSInstallationID:   s.VCSInstallationID,
		AccountLogin:        s.AccountLogin,
		AccountType:         s.AccountType,
		RepositorySelection: s.RepositorySelection,
		Permissions:         permissions,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}, nil
}

// GetVCSInstallation retrieves a VCS installation by internal ID using entityReader.
func (c *Client) GetVCSInstallation(ctx context.Context, id string) (*VCSInstallation, error) {
	r := newEntityReader[vcsInstallationMapper, spannerVCSInstallation, VCSInstallation, string](c)
	spannerInst, err := r.readRowByKey(ctx, id)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			return nil, ErrVCSInstallationNotFound
		}

		return nil, err
	}

	return spannerInst.toVCSInstallation()
}

// GetVCSInstallationByProviderID retrieves a VCS installation by provider and installation ID.
func (c *Client) GetVCSInstallationByProviderID(
	ctx context.Context,
	provider VCSProvider,
	vcsInstallationID string,
) (*VCSInstallation, error) {
	stmt := spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, AccountLogin, AccountType,
			RepositorySelection, Permissions, CreatedAt, UpdatedAt
		FROM VCSInstallations
		WHERE VCSProvider = @provider AND VCSInstallationID = @installationID`,
		Params: map[string]any{
			"provider":       string(provider),
			"installationID": vcsInstallationID,
		},
	}

	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrVCSInstallationNotFound
		}

		return nil, err
	}

	var spannerInst spannerVCSInstallation
	if err := row.ToStruct(&spannerInst); err != nil {
		return nil, err
	}

	return spannerInst.toVCSInstallation()
}

// UpsertVCSInstallation creates or updates a VCS installation in Spanner using entityWriter.
func (c *Client) UpsertVCSInstallation(ctx context.Context, in VCSInstallation) error {
	if in.ID == "" {
		existing, err := c.GetVCSInstallationByProviderID(ctx, in.VCSProvider, in.VCSInstallationID)
		if err != nil && !errors.Is(err, ErrVCSInstallationNotFound) {
			return err
		}
		if existing != nil {
			in.ID = existing.ID
			in.CreatedAt = existing.CreatedAt
		} else {
			in.ID = uuid.NewString()
		}
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = c.timeNow()
	}
	in.UpdatedAt = c.timeNow()

	return newEntityWriter[vcsInstallationMapper, VCSInstallation, spannerVCSInstallation, string](c).upsert(ctx, in)
}

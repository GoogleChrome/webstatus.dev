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
)

const vcsInstallationTable = "VCSInstallations"

var (
	// ErrVCSInstallationNotFound indicates that the installation record does not exist.
	ErrVCSInstallationNotFound = errors.New("vcs installation not found")
	// ErrUnknownVCSProvider indicates an unrecognized VCS provider.
	ErrUnknownVCSProvider = errors.New("unknown vcs provider")
)

// VCSProvider represents a supported version control system provider.
type VCSProvider string

const (
	VCSProviderGitHub VCSProvider = "github"
)

// GitHubAppPermissionLevel represents a GitHub App installation permission level.
type GitHubAppPermissionLevel string

const (
	GitHubAppPermissionLevelRead  GitHubAppPermissionLevel = "read"
	GitHubAppPermissionLevelWrite GitHubAppPermissionLevel = "write"
)

// Deprecated: use GitHubAppPermissionLevel instead.
type GitHubPermissionLevel = GitHubAppPermissionLevel

const (
	// Deprecated: use GitHubAppPermissionLevelRead instead.
	GitHubPermissionLevelRead GitHubAppPermissionLevel = GitHubAppPermissionLevelRead
	// Deprecated: use GitHubAppPermissionLevelWrite instead.
	GitHubPermissionLevelWrite GitHubAppPermissionLevel = GitHubAppPermissionLevelWrite
)

// GitHubPermissions represents GitHub App permission levels for an installation.
type GitHubPermissions struct {
	Issues       *GitHubAppPermissionLevel `json:"issues,omitempty"`
	Contents     *GitHubAppPermissionLevel `json:"contents,omitempty"`
	Metadata     *GitHubAppPermissionLevel `json:"metadata,omitempty"`
	PullRequests *GitHubAppPermissionLevel `json:"pull_requests,omitempty"`
	Workflows    *GitHubAppPermissionLevel `json:"workflows,omitempty"`
	Actions      *GitHubAppPermissionLevel `json:"actions,omitempty"`
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

type vcsInstallationKey struct {
	VCSProvider       VCSProvider
	VCSInstallationID string
}

type vcsInstallationMapper struct{}

func (m vcsInstallationMapper) Table() string { return vcsInstallationTable }

func (m vcsInstallationMapper) GetKeyFromExternal(in VCSInstallation) vcsInstallationKey {
	return vcsInstallationKey{
		VCSProvider:       in.VCSProvider,
		VCSInstallationID: in.VCSInstallationID,
	}
}

func (m vcsInstallationMapper) SelectOne(key vcsInstallationKey) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, AccountLogin, AccountType,
			RepositorySelection, Permissions, CreatedAt, UpdatedAt
		FROM VCSInstallations
		WHERE VCSProvider = @vcsProvider AND VCSInstallationID = @vcsInstallationID`,
		Params: map[string]any{
			"vcsProvider":       string(key.VCSProvider),
			"vcsInstallationID": key.VCSInstallationID,
		},
	}
}

func (m vcsInstallationMapper) GetID(key vcsInstallationKey) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID
		FROM VCSInstallations
		WHERE VCSProvider = @vcsProvider AND VCSInstallationID = @vcsInstallationID
		LIMIT 1`,
		Params: map[string]any{
			"vcsProvider":       string(key.VCSProvider),
			"vcsInstallationID": key.VCSInstallationID,
		},
	}
}

func (m vcsInstallationMapper) GetIDFromInternal(s spannerVCSInstallation) string {
	return s.ID
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

func (m vcsInstallationMapper) NewEntityWithID(in VCSInstallation) (spannerVCSInstallation, string, error) {
	id := in.ID
	if id == "" {
		id = uuid.NewString()
	}

	return spannerVCSInstallation{
		ID:                  id,
		VCSProvider:         string(in.VCSProvider),
		VCSInstallationID:   in.VCSInstallationID,
		AccountLogin:        in.AccountLogin,
		AccountType:         in.AccountType,
		RepositorySelection: in.RepositorySelection,
		Permissions: spanner.NullJSON{
			Value: extractPermissionsValue(in),
			Valid: true,
		},
		CreatedAt: spanner.CommitTimestamp,
		UpdatedAt: spanner.CommitTimestamp,
	}, id, nil
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
		UpdatedAt: spanner.CommitTimestamp,
	}
}

// ParseVCSProvider parses and validates a raw provider string into a typed VCSProvider.
func ParseVCSProvider(provider string) (VCSProvider, error) {
	p := VCSProvider(provider)
	switch p {
	case VCSProviderGitHub:
		return p, nil
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownVCSProvider, provider)
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
	provider, err := ParseVCSProvider(s.VCSProvider)
	if err != nil {
		return nil, err
	}
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

type vcsInstallationByIDMapper struct{}

func (m vcsInstallationByIDMapper) SelectOne(id string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSInstallationID, AccountLogin, AccountType,
			RepositorySelection, Permissions, CreatedAt, UpdatedAt
		FROM VCSInstallations
		WHERE ID = @id`,
		Params: map[string]any{"id": id},
	}
}

// GetVCSInstallation retrieves a VCS installation by internal ID.
func (c *Client) GetVCSInstallation(ctx context.Context, id string) (*VCSInstallation, error) {
	r := newEntityReader[vcsInstallationByIDMapper, spannerVCSInstallation, VCSInstallation, string](c)
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
	key := vcsInstallationKey{
		VCSProvider:       provider,
		VCSInstallationID: vcsInstallationID,
	}
	r := newEntityReader[vcsInstallationMapper, spannerVCSInstallation, VCSInstallation, vcsInstallationKey](c)
	spannerInst, err := r.readRowByKey(ctx, key)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			return nil, ErrVCSInstallationNotFound
		}

		return nil, err
	}

	return spannerInst.toVCSInstallation()
}

// UpsertVCSInstallation creates or updates a VCS installation in Spanner atomically using entityWriterWithIDRetrieval.
// It returns the ID of the installation.
func (c *Client) UpsertVCSInstallation(ctx context.Context, in VCSInstallation) (*string, error) {
	return newEntityWriterWithIDRetrieval[
		vcsInstallationMapper, string, VCSInstallation, spannerVCSInstallation, vcsInstallationKey,
	](c).upsertAndGetID(ctx, in)
}

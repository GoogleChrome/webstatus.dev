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
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ErrInvalidScanHistoryLimit indicates a non-positive per-repository scan history limit.
var ErrInvalidScanHistoryLimit = errors.New("per-repository scan history limit must be positive")

// RepositoryScanSummary is the part of a scan log row that vcs-sync needs to decide whether a
// repository should be scanned again.
//
// Status and error class values are returned as stored, without validation, so that one
// unexpected value cannot fail the whole lookup. Callers must treat unknown values conservatively.
type RepositoryScanSummary struct {
	ID         string
	ScanStatus ScanStatus
	ScannedAt  time.Time
	Branch     string
	CommitSHA  string
	// ErrorClass is only set for FAILED scans.
	ErrorClass *ScanErrorClass
	// RulesVersion is nil for rows written before the scanner recorded it.
	RulesVersion *int64
}

type spannerRepositoryScanSummary struct {
	ID           string             `spanner:"ID"`
	ScanStatus   string             `spanner:"ScanStatus"`
	ScannedAt    time.Time          `spanner:"ScannedAt"`
	Branch       string             `spanner:"Branch"`
	CommitSHA    string             `spanner:"CommitSHA"`
	ErrorClass   spanner.NullString `spanner:"ErrorClass"`
	RulesVersion spanner.NullInt64  `spanner:"RulesVersion"`
}

type spannerRepositoryScans struct {
	VCSRepositoryID string                          `spanner:"VCSRepositoryID"`
	Scans           []*spannerRepositoryScanSummary `spanner:"Scans"`
}

func (s *spannerRepositoryScanSummary) toRepositoryScanSummary() RepositoryScanSummary {
	var errClass *ScanErrorClass
	if s.ErrorClass.Valid {
		class := ScanErrorClass(s.ErrorClass.StringVal)
		errClass = &class
	}

	var rulesVersion *int64
	if s.RulesVersion.Valid {
		version := s.RulesVersion.Int64
		rulesVersion = &version
	}

	return RepositoryScanSummary{
		ID:           s.ID,
		ScanStatus:   ScanStatus(s.ScanStatus),
		ScannedAt:    s.ScannedAt,
		Branch:       s.Branch,
		CommitSHA:    s.CommitSHA,
		ErrorClass:   errClass,
		RulesVersion: rulesVersion,
	}
}

// GetRecentRepositoryScans returns the newest scan log rows of each repository, at most
// perRepoLimit per repository, newest first. Repositories without scan log rows are absent from
// the result.
func (c *Client) GetRecentRepositoryScans(
	ctx context.Context,
	vcsProvider VCSProvider,
	repoIDs []string,
	perRepoLimit int,
) (map[string][]RepositoryScanSummary, error) {
	if perRepoLimit <= 0 {
		return nil, ErrInvalidScanHistoryLimit
	}

	result := make(map[string][]RepositoryScanSummary, len(repoIDs))
	if len(repoIDs) == 0 {
		return result, nil
	}

	// Ordering by ScannedAt only matches the index, so each repository reads just its newest rows
	// instead of sorting its whole history. Ties are ordered by ID below.
	stmt := spanner.Statement{
		SQL: `SELECT repo AS VCSRepositoryID,
			ARRAY(
				SELECT AS STRUCT l.ID, l.ScanStatus, l.ScannedAt, l.Branch, l.CommitSHA,
					l.ErrorClass, l.RulesVersion
				FROM CodeSubscriptionScanLogs@{FORCE_INDEX=IDX_CodeSubscriptionScanLogs_Repo_ScannedAt} AS l
				WHERE l.VCSProvider = @vcsProvider AND l.VCSRepositoryID = repo
				ORDER BY l.ScannedAt DESC
				LIMIT @perRepoLimit
			) AS Scans
		FROM UNNEST(@repoIDs) AS repo`,
		Params: map[string]any{
			"vcsProvider":  string(vcsProvider),
			"repoIDs":      repoIDs,
			"perRepoLimit": int64(perRepoLimit),
		},
	}

	iter := c.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query recent repository scans: %w", err)
		}

		var repoScans spannerRepositoryScans
		if err := row.ToStruct(&repoScans); err != nil {
			return nil, fmt.Errorf("failed to parse recent repository scans: %w", err)
		}
		if len(repoScans.Scans) == 0 {
			continue
		}

		summaries := make([]RepositoryScanSummary, 0, len(repoScans.Scans))
		for _, scan := range repoScans.Scans {
			if scan != nil {
				summaries = append(summaries, scan.toRepositoryScanSummary())
			}
		}
		slices.SortFunc(summaries, func(a, b RepositoryScanSummary) int {
			if byTime := b.ScannedAt.Compare(a.ScannedAt); byTime != 0 {
				return byTime
			}

			return cmp.Compare(b.ID, a.ID)
		})
		result[repoScans.VCSRepositoryID] = summaries
	}

	return result, nil
}

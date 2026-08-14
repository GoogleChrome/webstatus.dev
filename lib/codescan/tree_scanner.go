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

package codescan

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxBlobSizeBytes  int64 = 1024 * 1024       // 1MB per file
	MaxScanTotalBytes int64 = 100 * 1024 * 1024 // 100MB total per repo
	MaxScannedFiles   int   = 10000             // 10k files max per scan
)

// ScanStatus represents the final status of a git tree scan in the domain.
type ScanStatus string

const (
	ScanStatusSuccess   ScanStatus = "SUCCESS"
	ScanStatusTruncated ScanStatus = "TRUNCATED"
	ScanStatusFailed    ScanStatus = "FAILED"
)

// SubscriptionOccurrence represents an individual location in code referencing a feature.
type SubscriptionOccurrence struct {
	FilePath       string
	LineNumber     int64
	CommentSnippet string
}

// ScannedSubscription represents an aggregated code subscription across occurrences.
type ScannedSubscription struct {
	ID                 string
	VCSProvider        string
	VCSInstallationID  string
	VCSRepositoryID    string
	RepositoryFullName string
	TargetQuery        string
	Triggers           []string
	Occurrences        []SubscriptionOccurrence
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// BlobReader abstracts content retrieval for git tree entries.
type BlobReader interface {
	GetBlob(ctx context.Context, repoFullName, sha string) ([]byte, error)
}

// GitTreeEntry represents a file entry in a VCS repository tree.
type GitTreeEntry struct {
	Path string
	Type string
	SHA  string
	Size int64
}

// ScanResult contains all extracted subscriptions and execution statistics.
type ScanResult struct {
	Subscriptions   []ScannedSubscription
	FilesScanned    int64
	DirectivesFound int64
	IsTruncated     bool
	ScanStatus      ScanStatus
}

type subscriptionGroupKey struct {
	TargetQuery string
	Trigger     SubscriptionTrigger
}

// IsSupportedWebExtension reports whether the file extension is on the strict web allowlist.
func IsSupportedWebExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs",
		".css", ".scss",
		".html", ".htm",
		".vue", ".svelte", ".astro":
		return true
	default:
		return false
	}
}

// SupportedWebExtensions returns a slice of all supported web file extensions.
func SupportedWebExtensions() []string {
	return []string{
		".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs",
		".css", ".scss",
		".html", ".htm",
		".vue", ".svelte", ".astro",
	}
}

func extractProjectDefaults(
	ctx context.Context,
	repoFullName string,
	treeEntries []GitTreeEntry,
	reader BlobReader,
) string {
	for _, entry := range treeEntries {
		if entry.Type != "blob" {
			continue
		}
		base := filepath.Base(entry.Path)
		if base == ".baseline.json" || base == "AGENTS.md" {
			content, err := reader.GetBlob(ctx, repoFullName, entry.SHA)
			if err == nil {
				defaultTarget, err := ParseProjectDefaults(content, base)
				if err == nil && defaultTarget != "" {
					return defaultTarget
				}
			}
		}
	}

	return ""
}

func shouldSkipEntry(entry GitTreeEntry) bool {
	if entry.Type != "blob" {
		return true
	}
	ext := filepath.Ext(entry.Path)

	return !IsSupportedWebExtension(ext) || entry.Size > MaxBlobSizeBytes
}

func buildSubscriptions(
	vcsProvider, installationID, repoID, repoFullName string,
	groups map[subscriptionGroupKey][]SubscriptionOccurrence,
	now time.Time,
) []ScannedSubscription {
	if vcsProvider == "" {
		vcsProvider = "github"
	}
	subscriptions := make([]ScannedSubscription, 0, len(groups))

	for key, occurrences := range groups {
		subID := uuid.New().String()
		subscriptions = append(subscriptions, ScannedSubscription{
			ID:                 subID,
			VCSProvider:        vcsProvider,
			VCSInstallationID:  installationID,
			VCSRepositoryID:    repoID,
			RepositoryFullName: repoFullName,
			TargetQuery:        key.TargetQuery,
			Triggers:           []string{string(key.Trigger)},
			Occurrences:        occurrences,
			CreatedAt:          now,
			UpdatedAt:          now,
		})
	}

	return subscriptions
}

// ScanGitTree walks tree entries, retrieves blobs, parses comment directives, and aggregates subscriptions.
func ScanGitTree(
	ctx context.Context,
	vcsProvider, installationID, repoID, repoFullName string,
	treeEntries []GitTreeEntry,
	reader BlobReader,
) (*ScanResult, error) {
	now := time.Now().UTC()
	var totalBytes int64
	scannedCount := 0
	isTruncated := false

	defaultTarget := extractProjectDefaults(ctx, repoFullName, treeEntries, reader)
	groups := make(map[subscriptionGroupKey][]SubscriptionOccurrence)

	for _, entry := range treeEntries {
		if shouldSkipEntry(entry) {
			continue
		}

		if scannedCount >= MaxScannedFiles || (totalBytes+entry.Size) > MaxScanTotalBytes {
			isTruncated = true

			break
		}

		content, err := reader.GetBlob(ctx, repoFullName, entry.SHA)
		if err != nil {
			continue
		}

		scannedCount++
		totalBytes += int64(len(content))

		directives := ParseFileDirectives(content, entry.Path, defaultTarget)
		for _, d := range directives {
			key := subscriptionGroupKey{
				TargetQuery: d.TargetQuery,
				Trigger:     d.Trigger,
			}
			groups[key] = append(groups[key], SubscriptionOccurrence{
				FilePath:       d.FilePath,
				LineNumber:     int64(d.LineNumber),
				CommentSnippet: d.RawSnippet,
			})
		}
	}

	subscriptions := buildSubscriptions(vcsProvider, installationID, repoID, repoFullName, groups, now)

	scanStatus := ScanStatusSuccess
	if isTruncated {
		scanStatus = ScanStatusTruncated
	}

	var directivesCount int64
	for _, sub := range subscriptions {
		directivesCount += int64(len(sub.Occurrences))
	}

	return &ScanResult{
		Subscriptions:   subscriptions,
		FilesScanned:    int64(scannedCount),
		DirectivesFound: directivesCount,
		IsTruncated:     isTruncated,
		ScanStatus:      scanStatus,
	}, nil
}

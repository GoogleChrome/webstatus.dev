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

// Package codescan provides AST parsing, directive scanning, and Git tree
// traversal for automated web platform feature code subscriptions.
package codescan

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// MaxBlobSizeBytes is the maximum allowed size for a single file blob (1MB).
	MaxBlobSizeBytes int64 = 1024 * 1024
	// MaxScanTotalBytes is the maximum aggregate byte budget per repository scan (100MB).
	MaxScanTotalBytes int64 = 100 * 1024 * 1024
	// MaxScannedFiles is the maximum number of source files scanned per repository.
	MaxScannedFiles int = 10000
	// MaxScanWarnings is the maximum number of diagnostic warnings retained per scan.
	MaxScanWarnings int = 100
)

// ScanStatus represents the final status of a git tree scan in the domain.
type ScanStatus string

const (
	// ScanStatusSuccess indicates that the tree scan completed successfully without hitting resource limits.
	ScanStatusSuccess ScanStatus = "SUCCESS"
	// ScanStatusTruncated indicates that the tree scan hit file count or byte limits and was truncated.
	ScanStatusTruncated ScanStatus = "TRUNCATED"
	// ScanStatusFailed indicates that the tree scan encountered an unrecoverable failure.
	ScanStatusFailed ScanStatus = "FAILED"
)

// SubscriptionOccurrence represents an individual location in source code referencing a web feature.
type SubscriptionOccurrence struct {
	FilePath       string
	LineNumber     int64
	CommentSnippet string
}

// ScannedSubscription represents an unpersisted code subscription extracted from source code AST comments.
// Note: It intentionally does not contain an ID, status, or timestamps, as primary keys and lifecycle
// are assigned by the persistence layer upon database synchronization.
type ScannedSubscription struct {
	VCSProvider        string
	VCSInstallationID  string
	VCSRepositoryID    string
	RepositoryFullName string
	TargetQuery        string
	Triggers           []SubscriptionTrigger
	Occurrences        []SubscriptionOccurrence
}

// BlobReader abstracts content retrieval for git tree entries from a VCS provider.
type BlobReader interface {
	// GetBlob retrieves the raw byte contents of a Git blob identified by its SHA.
	GetBlob(ctx context.Context, repoFullName, sha string) ([]byte, error)
}

// GitTreeEntry represents a file entry in a VCS repository tree.
type GitTreeEntry struct {
	Path string
	Type string
	SHA  string
	Size int64
}

// ScanResult contains all extracted subscriptions, execution statistics, and configuration warnings.
type ScanResult struct {
	Subscriptions   []ScannedSubscription
	FilesScanned    int64
	BytesScanned    int64
	DirectivesFound int64
	Warnings        []string
	IsTruncated     bool
	ScanStatus      ScanStatus
}

type groupedSubscription struct {
	TargetQuery string
	Triggers    map[SubscriptionTrigger]struct{}
	Occurrences []SubscriptionOccurrence
}

func appendWarning(warnings []string, warning string) []string {
	if len(warnings) < MaxScanWarnings {
		return append(warnings, warning)
	}
	if len(warnings) == MaxScanWarnings {
		return append(warnings, fmt.Sprintf(
			"maximum warning limit of %d reached; further warnings suppressed",
			MaxScanWarnings,
		))
	}

	return warnings
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

func extractBrowserslistConfig(
	ctx context.Context,
	repoFullName string,
	treeEntries []GitTreeEntry,
	reader BlobReader,
) (string, []string) {
	var target string
	var warnings []string

	for _, entry := range treeEntries {
		if err := ctx.Err(); err != nil {
			break
		}
		if entry.Type != "blob" || !IsBrowserslistFile(entry.Path) {
			continue
		}
		if entry.Size > MaxBlobSizeBytes {
			warnings = appendWarning(warnings, fmt.Sprintf(
				"browserslist file %q exceeds max blob size; skipped",
				entry.Path,
			))

			continue
		}
		content, err := reader.GetBlob(ctx, repoFullName, entry.SHA)
		if err != nil {
			warnings = appendWarning(warnings, fmt.Sprintf(
				"failed to read browserslist file %q (sha %s): %v",
				entry.Path, entry.SHA, err,
			))

			continue
		}
		if int64(len(content)) > MaxBlobSizeBytes {
			warnings = appendWarning(warnings, fmt.Sprintf(
				"browserslist file %q (sha %s) content exceeds max blob size; skipped",
				entry.Path, entry.SHA,
			))

			continue
		}
		t, w, err := ParseBrowserslistConfig(content, entry.Path)
		if err != nil {
			warnings = appendWarning(warnings, fmt.Sprintf(
				"failed to parse browserslist config in %q: %v",
				entry.Path, err,
			))

			continue
		}
		for _, warn := range w {
			warnings = appendWarning(warnings, warn)
		}
		if t != "" && target == "" {
			target = t
		}
	}

	return target, warnings
}

func shouldSkipEntry(entry GitTreeEntry) bool {
	if entry.Type != "blob" {
		return true
	}
	ext := filepath.Ext(entry.Path)

	return !IsSupportedWebExtension(ext)
}

// buildSubscriptions converts grouped subscriptions into ScannedSubscription objects.
// Note: Each ScannedSubscription represents a unique TargetQuery within the repository
// (deduplicated by the groups map). Internal IDs are assigned upon insertion in Spanner.
func buildSubscriptions(
	vcsProvider, installationID, repoID, repoFullName string,
	groups map[string]*groupedSubscription,
) []ScannedSubscription {
	if vcsProvider == "" {
		vcsProvider = "github"
	}
	subscriptions := make([]ScannedSubscription, 0, len(groups))

	for _, group := range groups {
		triggers := make([]SubscriptionTrigger, 0, len(group.Triggers))
		for t := range group.Triggers {
			triggers = append(triggers, t)
		}
		slices.Sort(triggers)

		// Sort occurrences deterministically by (FilePath, LineNumber)
		slices.SortFunc(group.Occurrences, func(a, b SubscriptionOccurrence) int {
			if c := strings.Compare(a.FilePath, b.FilePath); c != 0 {
				return c
			}

			return cmp.Compare(a.LineNumber, b.LineNumber)
		})

		subscriptions = append(subscriptions, ScannedSubscription{
			VCSProvider:        vcsProvider,
			VCSInstallationID:  installationID,
			VCSRepositoryID:    repoID,
			RepositoryFullName: repoFullName,
			TargetQuery:        group.TargetQuery,
			Triggers:           triggers,
			Occurrences:        group.Occurrences,
		})
	}

	// Sort subscriptions deterministically by TargetQuery
	slices.SortFunc(subscriptions, func(a, b ScannedSubscription) int {
		return strings.Compare(a.TargetQuery, b.TargetQuery)
	})

	return subscriptions
}

// ScanGitTree walks repository tree entries, retrieves supported web blobs, parses comment directives,
// and aggregates matching occurrences into code subscriptions. It enforces resource limits (max file count,
// per-blob size limits, total byte budget) and collects diagnostic warnings.
func ScanGitTree(
	ctx context.Context,
	vcsProvider, installationID, repoID, repoFullName string,
	treeEntries []GitTreeEntry,
	reader BlobReader,
) (*ScanResult, error) {
	var totalBytes int64
	scannedCount := 0
	isTruncated := false

	defaultTarget, warnings := extractBrowserslistConfig(ctx, repoFullName, treeEntries, reader)
	groups := make(map[string]*groupedSubscription)

	for _, entry := range treeEntries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if shouldSkipEntry(entry) {
			continue
		}

		// Pre-fetch per-blob size check
		if entry.Size > MaxBlobSizeBytes {
			warnings = appendWarning(warnings, fmt.Sprintf(
				"file %q exceeds max blob size (%d bytes > %d bytes); skipped",
				entry.Path, entry.Size, MaxBlobSizeBytes,
			))

			continue
		}

		// Pre-fetch total file limit check
		if scannedCount >= MaxScannedFiles {
			isTruncated = true
			warnings = appendWarning(warnings, fmt.Sprintf(
				"scan truncated: reached maximum file limit of %d files",
				MaxScannedFiles,
			))

			break
		}

		// Pre-fetch total bytes check when entry size metadata is available
		if entry.Size > 0 && totalBytes+entry.Size > MaxScanTotalBytes {
			isTruncated = true
			warnings = appendWarning(warnings, fmt.Sprintf(
				"scan truncated: total scanned bytes (%d + %d) would exceed limit of %d bytes",
				totalBytes, entry.Size, MaxScanTotalBytes,
			))

			break
		}

		content, err := reader.GetBlob(ctx, repoFullName, entry.SHA)
		if err != nil {
			warnings = appendWarning(warnings, fmt.Sprintf(
				"failed to retrieve blob for %q (sha %s): %v",
				entry.Path, entry.SHA, err,
			))

			continue
		}

		contentLen := int64(len(content))
		// Post-fetch size guard: protection against inaccurate or 0 entry.Size metadata
		if contentLen > MaxBlobSizeBytes {
			warnings = appendWarning(warnings, fmt.Sprintf(
				"file %q (sha %s) actual size exceeds max blob size (%d bytes > %d bytes); skipped",
				entry.Path, entry.SHA, contentLen, MaxBlobSizeBytes,
			))

			continue
		}

		// Post-fetch total bytes check using actual content length
		if totalBytes+contentLen > MaxScanTotalBytes {
			isTruncated = true
			warnings = appendWarning(warnings, fmt.Sprintf(
				"scan truncated: total scanned bytes would exceed limit of %d bytes",
				MaxScanTotalBytes,
			))

			break
		}

		scannedCount++
		totalBytes += contentLen

		defaultTrigger := SubscriptionTriggerFeatureBaselinePromoteToWidely
		if strings.EqualFold(defaultTarget, TargetNewly) {
			defaultTrigger = SubscriptionTriggerFeatureBaselinePromoteToNewly
		}

		directives, err := ParseReader(
			bytes.NewReader(content),
			entry.Path,
			defaultTrigger,
		)
		if err != nil {
			slog.WarnContext(ctx, "failed parsing file directives",
				"path", entry.Path,
				"error", err,
			)
			warnings = appendWarning(warnings, fmt.Sprintf("failed parsing %s: %v", entry.Path, err))

			continue
		}

		for _, d := range directives {
			targetKey := strings.ToLower(strings.TrimSpace(d.TargetQuery))
			group, exists := groups[targetKey]
			if !exists {
				group = &groupedSubscription{
					TargetQuery: targetKey,
					Triggers:    make(map[SubscriptionTrigger]struct{}),
					Occurrences: nil,
				}
				groups[targetKey] = group
			}
			group.Triggers[d.Trigger] = struct{}{}
			group.Occurrences = append(group.Occurrences, SubscriptionOccurrence{
				FilePath:       d.FilePath,
				LineNumber:     int64(d.LineNumber),
				CommentSnippet: d.RawSnippet,
			})
		}
	}

	subscriptions := buildSubscriptions(vcsProvider, installationID, repoID, repoFullName, groups)

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
		BytesScanned:    totalBytes,
		DirectivesFound: directivesCount,
		Warnings:        warnings,
		IsTruncated:     isTruncated,
		ScanStatus:      scanStatus,
	}, nil
}

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
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
)

const (
	codeSubscriptionScanLogTable = "CodeSubscriptionScanLogs"
	vcsWebhookDeliveryTable      = "VCSWebhookDeliveries"
)

var (
	// ErrScanLogNotFound indicates that the scan log record was not found.
	ErrScanLogNotFound = errors.New("scan log not found")
	// ErrVCSWebhookDeliveryNotFound indicates that the webhook delivery record was not found.
	ErrVCSWebhookDeliveryNotFound = errors.New("vcs webhook delivery not found")
	// ErrUnknownScanStatus indicates an unrecognized scan status.
	ErrUnknownScanStatus = errors.New("unknown scan status")
	// ErrUnknownScanErrorClass indicates an unrecognized scan error class.
	ErrUnknownScanErrorClass = errors.New("unknown scan error class")
)

// ScanStatus represents the completion status of an AST repository scan.
type ScanStatus string

const (
	ScanStatusSuccess   ScanStatus = "SUCCESS"
	ScanStatusTruncated ScanStatus = "TRUNCATED"
	ScanStatusFailed    ScanStatus = "FAILED"
)

// ScanErrorClass is a machine-readable category for a failed scan. It is used to decide
// whether a failure should page (see the vcs-scanner alerts) and to back off on repositories
// that keep failing.
type ScanErrorClass string

const (
	// ScanErrorClassPermanentClient is an unretryable 4XX response from GitHub (e.g. 403 blocked repo, 404).
	ScanErrorClassPermanentClient ScanErrorClass = "PERMANENT_CLIENT"
	// ScanErrorClassRedirectNotFollowed is a GitHub redirect that could not be followed.
	ScanErrorClassRedirectNotFollowed ScanErrorClass = "REDIRECT_NOT_FOLLOWED"
	// ScanErrorClassInvalidProvider is a task with an unsupported VCS provider.
	ScanErrorClassInvalidProvider ScanErrorClass = "INVALID_PROVIDER"
	// ScanErrorClassRateLimited is a GitHub primary or secondary rate limit response.
	ScanErrorClassRateLimited ScanErrorClass = "RATE_LIMITED"
	// ScanErrorClassTransientServer is a 5XX response from GitHub.
	ScanErrorClassTransientServer ScanErrorClass = "TRANSIENT_SERVER"
	// ScanErrorClassTimeout is a deadline or network timeout.
	ScanErrorClassTimeout ScanErrorClass = "TIMEOUT"
	// ScanErrorClassStall is an archive download aborted by the throughput watchdog.
	ScanErrorClassStall ScanErrorClass = "STALL"
	// ScanErrorClassStream is a broken connection or stream (reset, unexpected EOF, peer cancel).
	ScanErrorClassStream ScanErrorClass = "STREAM"
	// ScanErrorClassArchiveCorrupt is an archive that is not valid gzip/tar.
	ScanErrorClassArchiveCorrupt ScanErrorClass = "ARCHIVE_CORRUPT"
	// ScanErrorClassSpanner is a failure while writing scan results to Spanner.
	ScanErrorClassSpanner ScanErrorClass = "SPANNER"
	// ScanErrorClassPanic is a recovered panic while processing the scan.
	ScanErrorClassPanic ScanErrorClass = "PANIC"
	// ScanErrorClassCanceled is a scan canceled because the worker is shutting down.
	ScanErrorClassCanceled ScanErrorClass = "CANCELED"
	// ScanErrorClassUnknown is any failure that does not match a known category.
	ScanErrorClassUnknown ScanErrorClass = "UNKNOWN"
)

// ParseScanErrorClass parses and validates a raw scan error class string.
func ParseScanErrorClass(val string) (ScanErrorClass, error) {
	class := ScanErrorClass(val)
	switch class {
	case ScanErrorClassPermanentClient,
		ScanErrorClassRedirectNotFollowed,
		ScanErrorClassInvalidProvider,
		ScanErrorClassRateLimited,
		ScanErrorClassTransientServer,
		ScanErrorClassTimeout,
		ScanErrorClassStall,
		ScanErrorClassStream,
		ScanErrorClassArchiveCorrupt,
		ScanErrorClassSpanner,
		ScanErrorClassPanic,
		ScanErrorClassCanceled,
		ScanErrorClassUnknown:
		return class, nil
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownScanErrorClass, val)
}

// IsPermanent reports whether retrying the scan without a new push is not expected to help.
// Permanent classes are excluded from scan-failure alerts.
func (c ScanErrorClass) IsPermanent() bool {
	switch c {
	case ScanErrorClassPermanentClient,
		ScanErrorClassRedirectNotFollowed,
		ScanErrorClassInvalidProvider:
		return true
	case ScanErrorClassRateLimited,
		ScanErrorClassTransientServer,
		ScanErrorClassTimeout,
		ScanErrorClassStall,
		ScanErrorClassStream,
		ScanErrorClassArchiveCorrupt,
		ScanErrorClassSpanner,
		ScanErrorClassPanic,
		ScanErrorClassCanceled,
		ScanErrorClassUnknown:
		return false
	}

	return false
}

// CodeSubscriptionScanLog stores the audit log of an AST repository scan for a commit.
type CodeSubscriptionScanLog struct {
	ID              string
	VCSProvider     VCSProvider
	VCSRepositoryID string
	CommitSHA       string
	Branch          string
	ScanStatus      ScanStatus
	FilesScanned    int64
	DirectivesFound int64
	ErrorMessage    *string
	// ErrorClass is only set for FAILED scans.
	ErrorClass *ScanErrorClass
	// RulesVersion is the codescan.RulesVersion of the scanner that wrote the row. It is nil for
	// rows written before the scanner recorded it.
	RulesVersion *int64
	ScannedAt    time.Time
}

// spannerCodeSubscriptionScanLog is the internal struct for Spanner column mapping.
type spannerCodeSubscriptionScanLog struct {
	ID              string             `spanner:"ID"`
	VCSProvider     string             `spanner:"VCSProvider"`
	VCSRepositoryID string             `spanner:"VCSRepositoryID"`
	CommitSHA       string             `spanner:"CommitSHA"`
	Branch          string             `spanner:"Branch"`
	ScanStatus      string             `spanner:"ScanStatus"`
	FilesScanned    int64              `spanner:"FilesScanned"`
	DirectivesFound int64              `spanner:"DirectivesFound"`
	ErrorMessage    spanner.NullString `spanner:"ErrorMessage"`
	ErrorClass      spanner.NullString `spanner:"ErrorClass"`
	RulesVersion    spanner.NullInt64  `spanner:"RulesVersion"`
	ScannedAt       time.Time          `spanner:"ScannedAt"`
}

type codeSubscriptionScanLogMapper struct{}

func (m codeSubscriptionScanLogMapper) Table() string { return codeSubscriptionScanLogTable }

func (m codeSubscriptionScanLogMapper) SelectOne(id string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSRepositoryID, CommitSHA, Branch, ScanStatus,
			FilesScanned, DirectivesFound, ErrorMessage, ErrorClass, RulesVersion, ScannedAt
		FROM CodeSubscriptionScanLogs
		WHERE ID = @id`,
		Params: map[string]any{"id": id},
	}
}

func (m codeSubscriptionScanLogMapper) GetKeyFromExternal(in CodeSubscriptionScanLog) string {
	return in.ID
}

func (m codeSubscriptionScanLogMapper) NewEntity(
	id string,
	in CodeSubscriptionScanLog,
) (spannerCodeSubscriptionScanLog, error) {
	entityID := id
	if in.ID != "" {
		entityID = in.ID
	}

	entity := *fromCodeSubscriptionScanLog(&in)
	entity.ID = entityID

	return entity, nil
}

func (m codeSubscriptionScanLogMapper) Merge(
	in CodeSubscriptionScanLog,
	existing spannerCodeSubscriptionScanLog,
) spannerCodeSubscriptionScanLog {
	merged := *fromCodeSubscriptionScanLog(&in)
	merged.ID = existing.ID

	return merged
}

// ParseScanStatus parses and validates a raw scan status string into a typed ScanStatus.
func ParseScanStatus(val string) (ScanStatus, error) {
	status := ScanStatus(val)
	switch status {
	case ScanStatusSuccess,
		ScanStatusTruncated,
		ScanStatusFailed:
		return status, nil
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownScanStatus, val)
}

func (s *spannerCodeSubscriptionScanLog) toCodeSubscriptionScanLog() (*CodeSubscriptionScanLog, error) {
	provider, err := ParseVCSProvider(s.VCSProvider)
	if err != nil {
		return nil, err
	}
	status, err := ParseScanStatus(s.ScanStatus)
	if err != nil {
		return nil, err
	}

	var errMsg *string
	if s.ErrorMessage.Valid {
		errMsg = &s.ErrorMessage.StringVal
	}

	var errClass *ScanErrorClass
	if s.ErrorClass.Valid {
		parsed, err := ParseScanErrorClass(s.ErrorClass.StringVal)
		if err != nil {
			return nil, err
		}
		errClass = &parsed
	}

	var rulesVersion *int64
	if s.RulesVersion.Valid {
		rulesVersion = &s.RulesVersion.Int64
	}

	return &CodeSubscriptionScanLog{
		ID:              s.ID,
		VCSProvider:     provider,
		VCSRepositoryID: s.VCSRepositoryID,
		CommitSHA:       s.CommitSHA,
		Branch:          s.Branch,
		ScanStatus:      status,
		FilesScanned:    s.FilesScanned,
		DirectivesFound: s.DirectivesFound,
		ErrorMessage:    errMsg,
		ErrorClass:      errClass,
		RulesVersion:    rulesVersion,
		ScannedAt:       s.ScannedAt,
	}, nil
}

func fromCodeSubscriptionScanLog(log *CodeSubscriptionScanLog) *spannerCodeSubscriptionScanLog {
	var errMsg spanner.NullString
	if log.ErrorMessage != nil {
		errMsg = spanner.NullString{StringVal: *log.ErrorMessage, Valid: true}
	}

	var errClass spanner.NullString
	if log.ErrorClass != nil {
		errClass = spanner.NullString{StringVal: string(*log.ErrorClass), Valid: true}
	}

	var rulesVersion spanner.NullInt64
	if log.RulesVersion != nil {
		rulesVersion = spanner.NullInt64{Int64: *log.RulesVersion, Valid: true}
	}

	return &spannerCodeSubscriptionScanLog{
		ID:              log.ID,
		VCSProvider:     string(log.VCSProvider),
		VCSRepositoryID: log.VCSRepositoryID,
		CommitSHA:       log.CommitSHA,
		Branch:          log.Branch,
		ScanStatus:      string(log.ScanStatus),
		FilesScanned:    log.FilesScanned,
		DirectivesFound: log.DirectivesFound,
		ErrorMessage:    errMsg,
		ErrorClass:      errClass,
		RulesVersion:    rulesVersion,
		ScannedAt:       log.ScannedAt,
	}
}

// VCSWebhookDelivery represents an idempotency record for a received webhook delivery.
type VCSWebhookDelivery struct {
	VCSProvider     VCSProvider
	DeliveryGUID    string
	EventType       string
	VCSRepositoryID string
	ReceivedAt      time.Time
}

// spannerVCSWebhookDelivery is the internal struct for Spanner column mapping.
type spannerVCSWebhookDelivery struct {
	VCSProvider     string    `spanner:"VCSProvider"`
	DeliveryGUID    string    `spanner:"DeliveryGUID"`
	EventType       string    `spanner:"EventType"`
	VCSRepositoryID string    `spanner:"VCSRepositoryID"`
	ReceivedAt      time.Time `spanner:"ReceivedAt"`
}

type vcsWebhookDeliveryKey struct {
	VCSProvider  VCSProvider
	DeliveryGUID string
}

type vcsWebhookDeliveryMapper struct{}

func (m vcsWebhookDeliveryMapper) Table() string { return vcsWebhookDeliveryTable }

func (m vcsWebhookDeliveryMapper) SelectOne(key vcsWebhookDeliveryKey) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT VCSProvider, DeliveryGUID, EventType, VCSRepositoryID, ReceivedAt
		FROM VCSWebhookDeliveries
		WHERE VCSProvider = @vcsProvider AND DeliveryGUID = @deliveryGUID`,
		Params: map[string]any{
			"vcsProvider":  string(key.VCSProvider),
			"deliveryGUID": key.DeliveryGUID,
		},
	}
}

func (s *spannerVCSWebhookDelivery) toVCSWebhookDelivery() (*VCSWebhookDelivery, error) {
	provider, err := ParseVCSProvider(s.VCSProvider)
	if err != nil {
		return nil, err
	}

	return &VCSWebhookDelivery{
		VCSProvider:     provider,
		DeliveryGUID:    s.DeliveryGUID,
		EventType:       s.EventType,
		VCSRepositoryID: s.VCSRepositoryID,
		ReceivedAt:      s.ReceivedAt,
	}, nil
}

func fromVCSWebhookDelivery(d *VCSWebhookDelivery) *spannerVCSWebhookDelivery {
	return &spannerVCSWebhookDelivery{
		VCSProvider:     string(d.VCSProvider),
		DeliveryGUID:    d.DeliveryGUID,
		EventType:       d.EventType,
		VCSRepositoryID: d.VCSRepositoryID,
		ReceivedAt:      d.ReceivedAt,
	}
}

// GetCodeSubscriptionScanLog retrieves a scan log record by ID.
func (c *Client) GetCodeSubscriptionScanLog(
	ctx context.Context,
	id string,
) (*CodeSubscriptionScanLog, error) {
	r := newEntityReader[
		codeSubscriptionScanLogMapper, spannerCodeSubscriptionScanLog, CodeSubscriptionScanLog, string,
	](c)
	spannerLog, err := r.readRowByKey(ctx, id)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			return nil, ErrScanLogNotFound
		}

		return nil, err
	}

	return spannerLog.toCodeSubscriptionScanLog()
}

// GetVCSWebhookDelivery retrieves a webhook delivery idempotency record by provider and delivery GUID.
func (c *Client) GetVCSWebhookDelivery(
	ctx context.Context,
	provider VCSProvider,
	deliveryGUID string,
) (*VCSWebhookDelivery, error) {
	key := vcsWebhookDeliveryKey{
		VCSProvider:  provider,
		DeliveryGUID: deliveryGUID,
	}
	r := newEntityReader[
		vcsWebhookDeliveryMapper, spannerVCSWebhookDelivery, VCSWebhookDelivery, vcsWebhookDeliveryKey,
	](c)
	spannerDel, err := r.readRowByKey(ctx, key)
	if err != nil {
		if errors.Is(err, ErrQueryReturnedNoResults) {
			return nil, ErrVCSWebhookDeliveryNotFound
		}

		return nil, err
	}

	return spannerDel.toVCSWebhookDelivery()
}

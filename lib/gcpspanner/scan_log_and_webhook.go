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
)

// ScanStatus represents the completion status of an AST repository scan.
type ScanStatus string

const (
	ScanStatusSuccess   ScanStatus = "SUCCESS"
	ScanStatusTruncated ScanStatus = "TRUNCATED"
	ScanStatusFailed    ScanStatus = "FAILED"
)

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
	ScannedAt       time.Time
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
	ScannedAt       time.Time          `spanner:"ScannedAt"`
}

type codeSubscriptionScanLogMapper struct{}

func (m codeSubscriptionScanLogMapper) Table() string { return codeSubscriptionScanLogTable }

func (m codeSubscriptionScanLogMapper) SelectOne(id string) spanner.Statement {
	return spanner.Statement{
		SQL: `SELECT ID, VCSProvider, VCSRepositoryID, CommitSHA, Branch, ScanStatus,
			FilesScanned, DirectivesFound, ErrorMessage, ScannedAt
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
	var errMsg spanner.NullString
	if in.ErrorMessage != nil {
		errMsg = spanner.NullString{StringVal: *in.ErrorMessage, Valid: true}
	}

	entityID := id
	if in.ID != "" {
		entityID = in.ID
	}

	return spannerCodeSubscriptionScanLog{
		ID:              entityID,
		VCSProvider:     string(in.VCSProvider),
		VCSRepositoryID: in.VCSRepositoryID,
		CommitSHA:       in.CommitSHA,
		Branch:          in.Branch,
		ScanStatus:      string(in.ScanStatus),
		FilesScanned:    in.FilesScanned,
		DirectivesFound: in.DirectivesFound,
		ErrorMessage:    errMsg,
		ScannedAt:       in.ScannedAt,
	}, nil
}

func (m codeSubscriptionScanLogMapper) Merge(
	in CodeSubscriptionScanLog,
	existing spannerCodeSubscriptionScanLog,
) spannerCodeSubscriptionScanLog {
	var errMsg spanner.NullString
	if in.ErrorMessage != nil {
		errMsg = spanner.NullString{StringVal: *in.ErrorMessage, Valid: true}
	}

	return spannerCodeSubscriptionScanLog{
		ID:              existing.ID,
		VCSProvider:     string(in.VCSProvider),
		VCSRepositoryID: in.VCSRepositoryID,
		CommitSHA:       in.CommitSHA,
		Branch:          in.Branch,
		ScanStatus:      string(in.ScanStatus),
		FilesScanned:    in.FilesScanned,
		DirectivesFound: in.DirectivesFound,
		ErrorMessage:    errMsg,
		ScannedAt:       in.ScannedAt,
	}
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
		ScannedAt:       s.ScannedAt,
	}, nil
}

func fromCodeSubscriptionScanLog(log *CodeSubscriptionScanLog) *spannerCodeSubscriptionScanLog {
	var errMsg spanner.NullString
	if log.ErrorMessage != nil {
		errMsg = spanner.NullString{StringVal: *log.ErrorMessage, Valid: true}
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

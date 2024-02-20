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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
)

var ErrQueryReturnedNoResults = errors.New("query returned no results")
var ErrInternalQueryFailure = errors.New("internal spanner query failure")

// Client is the client for interacting with GCP Spanner.
type Client struct {
	*spanner.Client
}

// NewSpannerClient returns a Client for the Google Spanner service.
func NewSpannerClient(projectID string, instanceID string, name string) (*Client, error) {
	if projectID == "" {
		return nil, errors.New("projectID is empty")
	}
	if instanceID == "" {
		return nil, errors.New("instanceID is empty")
	}
	if name == "" {
		return nil, errors.New("name is empty")
	}

	client, err := spanner.NewClient(
		context.TODO(),
		fmt.Sprintf(
			"projects/%s/instances/%s/databases/%s",
			projectID, instanceID, name))
	if err != nil {
		return nil, err
	}

	return &Client{client}, nil
}

type Cursor struct {
	LastTimeStart time.Time `json:"last_time_start"`
	LastRunID     int64     `json:"last_run_id"`
}

func decodeCursor(cursor string) (Cursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return Cursor{}, err
	}
	var decoded Cursor
	err = json.Unmarshal(data, &decoded)

	return decoded, err
}

func encodeCursor(timeStart time.Time, id int64) string {
	cursor := Cursor{LastTimeStart: timeStart, LastRunID: id}
	data, err := json.Marshal(cursor)
	if err != nil {
		slog.Error("unable to encode cursor", "error", err)
	}

	return base64.RawURLEncoding.EncodeToString(data)
}

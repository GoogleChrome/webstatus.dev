// Copyright 2023 Google LLC
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

package gds

import (
	"context"
	"errors"
	"log/slog"

	"cloud.google.com/go/datastore"
)

const featureDataKey = "FeatureDataTest"

type Client struct {
	*datastore.Client
}

func NewDatastoreClient(projectID string, database *string) (*Client, error) {
	if projectID == "" {
		return nil, errors.New("projectID is empty")
	}
	if database == nil {
		return nil, errors.New("database is empty")
	}
	var err error
	var client *datastore.Client
	var databaseDB string
	if *database == "" {
		databaseDB = datastore.DefaultDatabaseID
	} else {
		databaseDB = *database
	}
	client, err = datastore.NewClientWithDatabase(context.TODO(), projectID, databaseDB)
	if err != nil {
		return nil, err
	}

	return &Client{client}, nil
}

type FeatureData struct {
	WebFeatureID string `datastore:"web_feature_id"`
	Name         string `datastore:"name"`
}

type Filterable interface {
	FilterQuery(*datastore.Query) *datastore.Query
}

type entityClient[T any] struct {
	*Client
}

func (c *entityClient[T]) upsert(ctx context.Context, kind string, data *T, filterables ...Filterable) error {
	// Begin a transaction.
	_, err := c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		// Get the entity, if it exists.
		var existingEntity []T
		query := datastore.NewQuery(kind)
		for _, filterable := range filterables {
			query = filterable.FilterQuery(query)
		}
		query = query.Limit(1).Transaction(tx)

		keys, err := c.GetAll(ctx, query, &existingEntity)
		if err != nil && !errors.Is(err, datastore.ErrNoSuchEntity) {
			slog.Error("unable to check for existing entities", "error", err)

			return err
		}

		var key *datastore.Key
		// If the entity exists, update it.
		if len(keys) > 0 {
			key = keys[0]

		} else {
			// If the entity does not exist, insert it.
			key = datastore.IncompleteKey(kind, nil)
		}

		_, err = tx.Put(key, data)
		if err != nil {
			// Handle any errors in an appropriate way, such as returning them.
			slog.Error("unable to upsert entity", "error", err)

			return err
		}

		return nil
	})

	if err != nil {
		slog.Error("failed to commit upsert transaction", "error", err)

		return err
	}

	return nil
}

func (c entityClient[T]) list(ctx context.Context, kind string, filterables ...Filterable) ([]*T, error) {
	var data []*T
	query := datastore.NewQuery(kind)
	for _, filterable := range filterables {
		query = filterable.FilterQuery(query)
	}
	_, err := c.GetAll(ctx, query, &data)
	if err != nil {
		slog.Error("failed to list data", "error", err, "kind", kind)

		return nil, err
	}

	return data, nil
}

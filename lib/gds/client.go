package gds

import (
	"context"
	"errors"
	"log/slog"

	"cloud.google.com/go/datastore"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
)

const featureDataKey = "FeatureDataTest"

type Client struct {
	*datastore.Client
}

func NewWebFeatureClient(projectID string, database *string) (*Client, error) {
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
	id           int64  // The integer ID used in the datastore.
}

func (c *Client) Upsert(
	ctx context.Context,
	webFeatureID string,
	featureData web_platform_dx__web_features.FeatureData,
) error {
	// Begin a transaction.
	_, err := c.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		// Get the entity, if it exists.
		var entity []FeatureData
		query := datastore.NewQuery(featureDataKey).FilterField("web_feature_id", "=", webFeatureID).Transaction(tx)

		keys, err := c.GetAll(ctx, query, &entity)
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
			key = datastore.IncompleteKey(featureDataKey, nil)
		}

		feature := &FeatureData{
			WebFeatureID: webFeatureID,
		}
		_, err = tx.Put(key, feature)
		if err != nil {
			// Handle any errors in an appropriate way, such as returning them.
			slog.Error("unable to upsert metadata", "error", err)
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

func (c *Client) List(ctx context.Context) ([]backend.Feature, error) {
	var featureData []*FeatureData
	_, err := c.GetAll(ctx, datastore.NewQuery(featureDataKey), &featureData)
	if err != nil {
		return nil, err
	}
	ret := make([]backend.Feature, len(featureData))
	for idx, val := range featureData {
		ret[idx] = backend.Feature{
			FeatureId: val.WebFeatureID,
		}
	}
	return ret, nil
}

func (c *Client) Get(ctx context.Context, webFeatureID string) (*backend.Feature, error) {
	var featureData []*FeatureData
	_, err := c.GetAll(
		ctx, datastore.NewQuery(featureDataKey).
			FilterField("web_feature_id", "=", webFeatureID),
		&featureData)
	if err != nil {
		return nil, err
	}
	return &backend.Feature{
		FeatureId: featureData[0].WebFeatureID,
	}, nil
}

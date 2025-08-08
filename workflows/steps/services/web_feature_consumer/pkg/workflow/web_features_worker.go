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

package workflow

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
)

// AssetGetter describes the behavior to get a certain asset from a github release.
type AssetGetter interface {
	DownloadFileFromRelease(
		ctx context.Context,
		owner, repo string,
		httpClient *http.Client,
		filePattern string) (io.ReadCloser, error)
}

// AssetParser describes the behavior to parse the io.ReadCloser from AssetGetter into the expected data type.
type AssetParser interface {
	Parse(io.ReadCloser) (*webdxfeaturetypes.ProcessedWebFeaturesData, error)
}

// WebFeatureStorer describes the logic to insert the web features that were returned by the AssetParser.
type WebFeatureStorer interface {
	InsertWebFeatures(
		ctx context.Context,
		data map[string]web_platform_dx__web_features.FeatureValue,
		startAt time.Time, endAt time.Time) (map[string]string, error)
	InsertMovedWebFeatures(
		ctx context.Context,
		data map[string]web_platform_dx__web_features.FeatureMovedData,
	) error
	InsertSplitWebFeatures(
		ctx context.Context,
		data map[string]web_platform_dx__web_features.FeatureSplitData,
	) error
}

// WebFeatureMetadataStorer describes the logic to insert the non-relation metadata about web features that
// were returned by the AssetParser.
type WebFeatureMetadataStorer interface {
	InsertWebFeaturesMetadata(
		ctx context.Context,
		featureKeyToID map[string]string,
		data map[string]web_platform_dx__web_features.FeatureValue) error
}

// WebDXGroupStorer describes the logic to insert the groups that were returned by the AssetParser.
type WebDXGroupStorer interface {
	InsertWebFeatureGroups(
		ctx context.Context,
		featureData map[string]web_platform_dx__web_features.FeatureValue,
		groupData map[string]web_platform_dx__web_features.GroupData) error
}

// WebDXSnapshotStorer describes the logic to insert the snapshots that were returned by the AssetParser.
type WebDXSnapshotStorer interface {
	InsertWebFeatureSnapshots(
		ctx context.Context,
		featureKeyToID map[string]string,
		featureData map[string]web_platform_dx__web_features.FeatureValue,
		snapshotData map[string]web_platform_dx__web_features.SnapshotData) error
}

func NewWebFeaturesJobProcessor(assetGetter AssetGetter,
	storer WebFeatureStorer,
	metadataStorer WebFeatureMetadataStorer,
	groupStorer WebDXGroupStorer,
	snapshotStorer WebDXSnapshotStorer,
	webFeaturesDataParser AssetParser,
) WebFeaturesJobProcessor {
	return WebFeaturesJobProcessor{
		assetGetter:           assetGetter,
		storer:                storer,
		metadataStorer:        metadataStorer,
		groupStorer:           groupStorer,
		snapshotStorer:        snapshotStorer,
		webFeaturesDataParser: webFeaturesDataParser,
	}
}

type WebFeaturesJobProcessor struct {
	assetGetter           AssetGetter
	storer                WebFeatureStorer
	metadataStorer        WebFeatureMetadataStorer
	groupStorer           WebDXGroupStorer
	snapshotStorer        WebDXSnapshotStorer
	webFeaturesDataParser AssetParser
}

func (p WebFeaturesJobProcessor) Process(ctx context.Context, job JobArguments) error {
	file, err := p.assetGetter.DownloadFileFromRelease(
		ctx,
		job.repoOwner,
		job.repoName,
		http.DefaultClient,
		job.assetName)
	if err != nil {
		slog.ErrorContext(ctx, "unable to get asset", "error", err)

		return err
	}

	data, err := p.webFeaturesDataParser.Parse(file)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse data", "error", err)

		return err
	}

	mapping, err := p.storer.InsertWebFeatures(ctx, data.Features.Data, job.startAt, job.endAt)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store data", "error", err)

		return err
	}

	err = p.metadataStorer.InsertWebFeaturesMetadata(ctx, mapping, data.Features.Data)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store metadata", "error", err)

		return err
	}

	err = p.groupStorer.InsertWebFeatureGroups(ctx, data.Features.Data, data.Groups)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store groups", "error", err)

		return err
	}

	err = p.snapshotStorer.InsertWebFeatureSnapshots(ctx, mapping, data.Features.Data, data.Snapshots)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store snapshots", "error", err)

		return err
	}

	err = p.storer.InsertMovedWebFeatures(ctx, data.Features.Moved)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store moved features", "error", err)

		return err
	}

	err = p.storer.InsertSplitWebFeatures(ctx, data.Features.Split)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store split features", "error", err)

		return err
	}

	return nil
}

func NewJobArguments(assetName, repoOwner, repoName string, startAt, endAt time.Time) JobArguments {
	return JobArguments{
		assetName: assetName,
		repoOwner: repoOwner,
		repoName:  repoName,
		startAt:   startAt,
		endAt:     endAt,
	}
}

type JobArguments struct {
	assetName string // Asset Name in Github Release
	repoOwner string
	repoName  string
	startAt   time.Time
	endAt     time.Time
}

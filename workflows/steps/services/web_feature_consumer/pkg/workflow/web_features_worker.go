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
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gh"
	"github.com/GoogleChrome/webstatus.dev/lib/webdxfeaturetypes"
	"golang.org/x/mod/semver"
)

// AssetGetter describes the behavior to get a certain asset from a github release.
type AssetGetter interface {
	DownloadFileFromRelease(
		ctx context.Context,
		owner, repo string,
		httpClient *http.Client,
		filePattern string) (*gh.ReleaseFile, error)
}

// AssetParser describes the behavior to parse the io.ReadCloser from AssetGetter into the expected data type.
type AssetParser interface {
	Parse(io.ReadCloser) (*webdxfeaturetypes.ProcessedWebFeaturesData, error)
}

var (
	ErrUnknownAssetVersion     = errors.New("unknown asset version")
	ErrUnsupportedAssetVersion = errors.New("unsupported asset version")
)

// WebFeatureStorer describes the logic to insert the web features that were returned by the AssetParser.
type WebFeatureStorer interface {
	InsertWebFeatures(
		ctx context.Context,
		data *webdxfeaturetypes.ProcessedWebFeaturesData,
		startAt time.Time, endAt time.Time) (map[string]string, error)
	InsertMovedWebFeatures(
		ctx context.Context,
		data map[string]webdxfeaturetypes.FeatureMovedData,
	) error
	InsertSplitWebFeatures(
		ctx context.Context,
		data map[string]webdxfeaturetypes.FeatureSplitData,
	) error
}

// WebFeatureMetadataStorer describes the logic to insert the non-relation metadata about web features that
// were returned by the AssetParser.
type WebFeatureMetadataStorer interface {
	InsertWebFeaturesMetadata(
		ctx context.Context,
		featureKeyToID map[string]string,
		data map[string]webdxfeaturetypes.FeatureValue) error
}

// WebDXGroupStorer describes the logic to insert the groups that were returned by the AssetParser.
type WebDXGroupStorer interface {
	InsertWebFeatureGroups(
		ctx context.Context,
		featureData map[string]webdxfeaturetypes.FeatureValue,
		groupData map[string]webdxfeaturetypes.GroupData) error
}

// WebDXSnapshotStorer describes the logic to insert the snapshots that were returned by the AssetParser.
type WebDXSnapshotStorer interface {
	InsertWebFeatureSnapshots(
		ctx context.Context,
		featureKeyToID map[string]string,
		featureData map[string]webdxfeaturetypes.FeatureValue,
		snapshotData map[string]webdxfeaturetypes.SnapshotData) error
}

func NewWebFeaturesJobProcessor(assetGetter AssetGetter,
	storer WebFeatureStorer,
	metadataStorer WebFeatureMetadataStorer,
	groupStorer WebDXGroupStorer,
	snapshotStorer WebDXSnapshotStorer,
	webFeaturesDataV2Parser AssetParser,
	webFeaturesDataV3Parser AssetParser,
) WebFeaturesJobProcessor {
	return WebFeaturesJobProcessor{
		assetGetter:             assetGetter,
		storer:                  storer,
		metadataStorer:          metadataStorer,
		groupStorer:             groupStorer,
		snapshotStorer:          snapshotStorer,
		webFeaturesDataV2Parser: webFeaturesDataV2Parser,
		webFeaturesDataV3Parser: webFeaturesDataV3Parser,
	}
}

type WebFeaturesJobProcessor struct {
	assetGetter             AssetGetter
	storer                  WebFeatureStorer
	metadataStorer          WebFeatureMetadataStorer
	groupStorer             WebDXGroupStorer
	snapshotStorer          WebDXSnapshotStorer
	webFeaturesDataV2Parser AssetParser
	webFeaturesDataV3Parser AssetParser
}

const (
	// According to https://pkg.go.dev/golang.org/x/mod/semver, the version must start with "v".
	v2 = "v2.0.0"
	v3 = "v3.0.0"
	v4 = "v4.0.0"
)

func (p WebFeaturesJobProcessor) parseByVersion(ctx context.Context, file *gh.ReleaseFile) (
	*webdxfeaturetypes.ProcessedWebFeaturesData, error) {
	if file.Info.Tag == nil {
		slog.ErrorContext(ctx, "unknown version", "version", "nil")

		return nil, ErrUnknownAssetVersion
	}

	if semver.Compare(*file.Info.Tag, v3) == -1 {
		// If less than version 3, use default v2 parser
		slog.InfoContext(ctx, "using v2 parser", "version", *file.Info.Tag)
		data, err := p.webFeaturesDataV2Parser.Parse(file.Contents)
		if err != nil {
			slog.ErrorContext(ctx, "unable to parse v2 data", "error", err)

			return nil, err
		}

		return data, nil

	} else if semver.Compare(*file.Info.Tag, v4) == -1 {
		// If version 3, use v3 parser
		slog.InfoContext(ctx, "using v3 parser", "version", *file.Info.Tag)
		data, err := p.webFeaturesDataV3Parser.Parse(file.Contents)
		if err != nil {
			slog.ErrorContext(ctx, "unable to parse v3 data", "error", err)

			return nil, err
		}

		return data, nil
	}

	slog.ErrorContext(ctx, "unsupported version", "version", *file.Info.Tag)

	return nil, ErrUnsupportedAssetVersion
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

	data, err := p.parseByVersion(ctx, file)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse data", "error", err)

		return err
	}

	mapping, err := p.storer.InsertWebFeatures(ctx, data, job.startAt, job.endAt)
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

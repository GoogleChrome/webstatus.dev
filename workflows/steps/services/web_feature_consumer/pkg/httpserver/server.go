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

package httpserver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/web_feature_consumer"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/web_feature_consumer/pkg/data"
	"github.com/go-chi/chi/v5"
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
	Parse(io.ReadCloser) (*web_platform_dx__web_features.FeatureData, error)
}

// WebFeatureStorer describes the logic to insert the web features that were returned by the AssetParser.
type WebFeatureStorer interface {
	InsertWebFeatures(
		ctx context.Context,
		data map[string]web_platform_dx__web_features.FeatureValue) (map[string]string, error)
}

// WebFeatureMetadataStorer describes the logic to insert the non-relation metadata about web features that
// were returned by the AssetParser.
type WebFeatureMetadataStorer interface {
	InsertWebFeaturesMetadata(
		ctx context.Context,
		featureKeyToID map[string]string,
		data map[string]web_platform_dx__web_features.FeatureValue) error
}

type Server struct {
	assetGetter           AssetGetter
	storer                WebFeatureStorer
	metadataStorer        WebFeatureMetadataStorer
	webFeaturesDataParser AssetParser
	defaultAssetName      string
	defaultRepoOwner      string
	defaultRepoName       string
}

// PostV1WebFeatures implements web_feature_consumer.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) PostV1WebFeatures(
	ctx context.Context,
	_ web_feature_consumer.PostV1WebFeaturesRequestObject,
) (web_feature_consumer.PostV1WebFeaturesResponseObject, error) {
	file, err := s.assetGetter.DownloadFileFromRelease(
		ctx,
		s.defaultRepoOwner,
		s.defaultRepoName,
		http.DefaultClient,
		s.defaultAssetName)
	if err != nil {
		slog.ErrorContext(ctx, "unable to get asset", "error", err)

		return web_feature_consumer.PostV1WebFeatures500JSONResponse{
			Code:    500,
			Message: "unable to get asset",
		}, nil
	}

	data, err := s.webFeaturesDataParser.Parse(file)
	if err != nil {
		slog.ErrorContext(ctx, "unable to parse data", "error", err)

		return web_feature_consumer.PostV1WebFeatures500JSONResponse{
			Code:    500,
			Message: "unable to parse data",
		}, nil
	}

	mapping, err := s.storer.InsertWebFeatures(ctx, data.Features)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store data", "error", err)

		return web_feature_consumer.PostV1WebFeatures500JSONResponse{
			Code:    500,
			Message: "unable to store data",
		}, nil
	}

	err = s.metadataStorer.InsertWebFeaturesMetadata(ctx, mapping, data.Features)
	if err != nil {
		slog.ErrorContext(ctx, "unable to store metadata", "error", err)

		return web_feature_consumer.PostV1WebFeatures500JSONResponse{
			Code:    500,
			Message: "unable to store metadata",
		}, nil
	}

	return web_feature_consumer.PostV1WebFeatures200Response{}, nil
}

func NewHTTPServer(
	port string,
	assetGetter AssetGetter,
	storer WebFeatureStorer,
	metadataStorer WebFeatureMetadataStorer,
	defaultAssetName string,
	defaultRepoOwner string,
	defaultRepoName string,
) (*http.Server, error) {
	_, err := web_feature_consumer.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("error loading swagger spec. %w", err)
	}

	// Create an instance of our handler which satisfies the generated interface
	srv := &Server{
		assetGetter:           assetGetter,
		storer:                storer,
		metadataStorer:        metadataStorer,
		webFeaturesDataParser: data.Parser{},
		defaultAssetName:      defaultAssetName,
		defaultRepoOwner:      defaultRepoOwner,
		defaultRepoName:       defaultRepoName,
	}

	srvStrictHandler := web_feature_consumer.NewStrictHandler(srv, nil)

	// This is how you set up a basic chi router
	r := chi.NewRouter()

	// Use our validation middleware to check all requests against the
	// web_feature_consumer schema.
	// r.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
	// 	SilenceServersWarning: true,
	// }))

	// We now register our web feature router above as the handler for the interface
	web_feature_consumer.HandlerFromMux(srvStrictHandler, r)

	// nolint:exhaustruct // No need to populate 3rd party struct
	return &http.Server{
		Handler:           r,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 30 * time.Second,
	}, nil
}

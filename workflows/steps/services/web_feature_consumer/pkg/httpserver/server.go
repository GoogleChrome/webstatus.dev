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
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/jsonschema/web_platform_dx__web_features"
	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/web_feature_consumer"
	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/yaml"
)

type ObjectGetter interface {
	Get(ctx context.Context, filename string) ([]byte, error)
}

type WebFeatureMetadataStorer interface {
	Upsert(ctx context.Context, webFeatureID string, featureData web_platform_dx__web_features.FeatureData) error
}

type Server struct {
	objectGetter   ObjectGetter
	metadataStorer WebFeatureMetadataStorer
}

func getFilenameBaseWithoutExt(filePath string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)

	return strings.TrimSuffix(base, ext)
}

// PostV1WebFeatures implements web_feature_consumer.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) PostV1WebFeatures(
	ctx context.Context,
	request web_feature_consumer.PostV1WebFeaturesRequestObject,
) (web_feature_consumer.PostV1WebFeaturesResponseObject, error) {
	webFeatureKey := getFilenameBaseWithoutExt(request.Body.Location.Gcs.Object)
	// TODO allow input to configure the bucket it looks into.
	yamlBytes, err := s.objectGetter.Get(ctx, request.Body.Location.Gcs.Object)
	if err != nil {
		// TODO check error type
		slog.Error("unable to get file", "file", request.Body.Location.Gcs.Object, "error", err)

		return web_feature_consumer.PostV1WebFeatures404JSONResponse{
			Code:    404,
			Message: "unable to get file",
		}, nil
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		slog.Error("unable to read data", "error", err)

		return web_feature_consumer.PostV1WebFeatures400JSONResponse{
			Code:    400,
			Message: "unable to read file as json",
		}, nil
	}
	featureData, err := web_platform_dx__web_features.UnmarshalFeatureData(jsonBytes)
	if err != nil {
		slog.Error("unable to convert data", "error", err, "bytes", string(jsonBytes))

		return web_feature_consumer.PostV1WebFeatures500JSONResponse{
			Code:    500,
			Message: "unable to convert data to expected format",
		}, nil
	}

	err = s.metadataStorer.Upsert(ctx, webFeatureKey, featureData)
	if err != nil {
		slog.Error("unable to store data", "error", err)

		return web_feature_consumer.PostV1WebFeatures400JSONResponse{
			Code:    400,
			Message: "unable to store data",
		}, nil
	}

	return web_feature_consumer.PostV1WebFeatures200Response{}, nil
}

func NewHTTPServer(
	port string,
	objectGetter ObjectGetter,
	metadataStorer WebFeatureMetadataStorer,
) (*http.Server, error) {
	_, err := web_feature_consumer.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("error loading swagger spec. %w", err)
	}

	// Create an instance of our handler which satisfies the generated interface
	srv := &Server{
		objectGetter:   objectGetter,
		metadataStorer: metadataStorer,
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

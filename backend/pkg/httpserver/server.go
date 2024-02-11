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
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type WebFeatureMetadataStorer interface {
	ListWebFeataureData(ctx context.Context, nextPageToken *string) ([]backend.Feature, *string, error)
	GetWebFeatureData(ctx context.Context, featureID string) (*backend.Feature, error)
}

type Server struct {
	metadataStorer WebFeatureMetadataStorer
}

// GetV1FeaturesFeatureId implements backend.StrictServerInterface.
// nolint: revive, ireturn // Name generated from openapi
func (s *Server) GetV1FeaturesFeatureId(
	ctx context.Context,
	request backend.GetV1FeaturesFeatureIdRequestObject,
) (backend.GetV1FeaturesFeatureIdResponseObject, error) {
	feature, err := s.metadataStorer.GetWebFeatureData(ctx, request.FeatureId)
	if err != nil {
		slog.Error("unable to get feature", "error", err)

		return backend.GetV1FeaturesFeatureId500JSONResponse{
			Code:    500,
			Message: "unable to get feature",
		}, nil
	}

	return backend.GetV1FeaturesFeatureId200JSONResponse(*feature), nil
}

// GetV1Features implements backend.StrictServerInterface.
// nolint:ireturn // Expected ireturn for openapi generation.
func (s *Server) GetV1Features(
	ctx context.Context,
	_ backend.GetV1FeaturesRequestObject,
) (backend.GetV1FeaturesResponseObject, error) {
	featureData, _, err := s.metadataStorer.ListWebFeataureData(ctx, nil)
	if err != nil {
		// TODO check error type
		slog.Error("unable to get list of features", "error", err)

		return backend.GetV1Features500JSONResponse{
			Code:    500,
			Message: "unable to get list of features",
		}, nil
	}

	return backend.GetV1Features200JSONResponse{
		Metadata: nil,
		Data:     featureData,
	}, nil
}

func NewHTTPServer(port string, metadataStorer WebFeatureMetadataStorer, allowedOrigin string) (*http.Server, error) {
	_, err := backend.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("error loading swagger spec. %w", err)
	}

	// Create an instance of our handler which satisfies the generated interface
	srv := &Server{
		metadataStorer: metadataStorer,
	}

	srvStrictHandler := backend.NewStrictHandler(srv, nil)

	// This is how you set up a basic chi router
	r := chi.NewRouter()
	//nolint: exhaustruct // No need to use every option of 3rd party struct.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{allowedOrigin},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods: []string{"GET", "OPTIONS"},
		// AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		// ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Remove after UbP
		MaxAge:           300,  // Maximum value not ignored by any of major browsers
	}))

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	// r.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
	// 	SilenceServersWarning: true,
	// }))

	// We now register our web feature router above as the handler for the interface
	backend.HandlerFromMux(srvStrictHandler, r)

	// nolint:exhaustruct // No need to populate 3rd party struct
	return &http.Server{
		Handler:           r,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 30 * time.Second,
	}, nil
}

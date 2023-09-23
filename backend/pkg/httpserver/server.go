package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type WebFeatureMetadataStorer interface {
	List(ctx context.Context) ([]backend.Feature, error)
}

type Server struct {
	metadataStorer WebFeatureMetadataStorer
}

// GetV1Features implements backend.StrictServerInterface.
func (s *Server) GetV1Features(
	ctx context.Context,
	request backend.GetV1FeaturesRequestObject,
) (backend.GetV1FeaturesResponseObject, error) {
	featureData, err := s.metadataStorer.List(ctx)
	if err != nil {
		// TODO check error type
		slog.Error("unable to get list of features", "error", err)
		return backend.GetV1Features500JSONResponse{
			Code:    500,
			Message: "unable to get list of features",
		}, nil
	}
	return backend.GetV1Features200JSONResponse{
		Data: featureData,
	}, nil
}

func NewHTTPServer(port string, metadataStorer WebFeatureMetadataStorer) (*http.Server, error) {
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
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods: []string{"GET", "OPTIONS"},
		// AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		// ExposedHeaders:   []string{"Link"},
		// AllowCredentials: false,
		MaxAge: 300, // Maximum value not ignored by any of major browsers
	}))

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	// r.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
	// 	SilenceServersWarning: true,
	// }))

	// We now register our web feature router above as the handler for the interface
	backend.HandlerFromMux(srvStrictHandler, r)

	return &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort("0.0.0.0", port),
	}, nil
}

package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/backend"
	"github.com/go-chi/chi/v5"
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

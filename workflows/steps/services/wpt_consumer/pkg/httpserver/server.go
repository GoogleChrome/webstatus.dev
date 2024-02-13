package httpserver

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/wpt_consumer"
	"github.com/go-chi/chi/v5"
)

type WorkflowStarter interface {
	Start(ctx context.Context, from time.Time) error
}

type Server struct {
	workflowStarter WorkflowStarter
	from            time.Time
}

// PostV1Wpt implements wpt_consumer.StrictServerInterface.
// nolint: revive, ireturn // Signature generated from openapi
func (s *Server) PostV1Wpt(
	ctx context.Context,
	request wpt_consumer.PostV1WptRequestObject) (wpt_consumer.PostV1WptResponseObject, error) {
	err := s.workflowStarter.Start(ctx, s.from)
	if err != nil {
		slog.Error("workflow failed", "error", err)

		return wpt_consumer.PostV1Wpt500JSONResponse{
			Code:    500,
			Message: "workflow failed",
		}, nil
	}

	return wpt_consumer.PostV1Wpt200Response{}, nil
}

func NewHTTPServer(
	port string,
	workflowStarter WorkflowStarter,
	from time.Time,
) (*http.Server, error) {
	_, err := wpt_consumer.GetSwagger()
	if err != nil {
		return nil, err
	}
	srv := &Server{
		workflowStarter: workflowStarter,
		from:            from,
	}

	handler := wpt_consumer.NewStrictHandler(srv, nil)

	r := chi.NewRouter()

	// We now register our wpt consumer router above as the handler for the interface
	wpt_consumer.HandlerFromMux(handler, r)

	// nolint:exhaustruct // No need to populate 3rd party struct
	return &http.Server{
		Handler:           r,
		Addr:              net.JoinHostPort("0.0.0.0", port),
		ReadHeaderTimeout: 30 * time.Second,
	}, nil
}

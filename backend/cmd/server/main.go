package main

import (
	"log/slog"
	"os"

	"github.com/GoogleChrome/webstatus.dev/backend/pkg/httpserver"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
)

func main() {
	var datastoreDB *string
	if value, found := os.LookupEnv("DATASTORE_DATABASE"); found {
		datastoreDB = &value
	}
	fs, err := gds.NewWebFeatureClient(os.Getenv("PROJECT_ID"), datastoreDB)
	if err != nil {
		slog.Error("failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}

	srv, err := httpserver.NewHTTPServer("8080", fs)
	if err != nil {
		slog.Error("unable to create server", "error", err.Error())
		os.Exit(1)
	}
	err = srv.ListenAndServe()
	if err != nil {
		slog.Error("unable to start server", "error", err.Error())
		os.Exit(1)
	}
}

package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"cloud.google.com/go/storage"
	"github.com/GoogleChrome/webstatus.dev/lib/gcs"
	"github.com/GoogleChrome/webstatus.dev/lib/gds"
	"github.com/GoogleChrome/webstatus.dev/workflows/steps/services/web_feature_consumer/pkg/httpserver"
	"google.golang.org/api/option"
)

func main() {

	bucket := os.Getenv("BUCKET")
	storageEmulator := os.Getenv("STORAGE_EMULATOR_HOST")
	var options []option.ClientOption
	if storageEmulator != "" {
		slog.Info("found storage emulator")
		options = append(options, option.WithEndpoint(storageEmulator+"/storage/v1/"))
	}
	client, err := storage.NewClient(context.TODO(), options...)
	if err != nil {
		log.Fatalf("failed to create base client: %v", err)
	}
	gcsObjectGetter, err := gcs.NewClient(client, bucket)
	if err != nil {
		slog.Error("failed to create client", "error", err.Error())
		os.Exit(1)
	}
	var datastoreDB *string
	if value, found := os.LookupEnv("DATASTORE_DATABASE"); found {
		datastoreDB = &value
	}
	fs, err := gds.NewWebFeatureClient(os.Getenv("PROJECT_ID"), datastoreDB)
	if err != nil {
		slog.Error("failed to create datastore client", "error", err.Error())
		os.Exit(1)
	}

	srv, err := httpserver.NewHTTPServer("8080", gcsObjectGetter, fs)
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

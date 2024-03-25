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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/GoogleChrome/webstatus.dev/lib/gen/openapi/workflows/steps/web_feature_consumer"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Describe the command line flags and parse the flags
	var (
		webFeatureConsumerHost = flag.String("web_consumer_host", "", "Web Feature Consumer host")
		githubOrg              = flag.String("github_org", "web-platform-dx", "Github Org")
		githubRepo             = flag.String("github_repo", "web-features", "Github Repo")
	)
	flag.Parse()

	webFeatureConsumerClient, err := web_feature_consumer.NewClientWithResponses(*webFeatureConsumerHost)
	if err != nil {
		log.Fatalf("failed to construct repo downloader client: %s\n", err.Error())
	}

	// Run the workflow
	err = newWebFeatureWorkflow(
		webFeatureConsumerClient,
		*githubOrg,
		*githubRepo,
	).Run(context.Background())
	if err != nil {
		log.Fatalf("failed to run web feature workflow: %s\n", err.Error())
	}

	log.Println("web feature workflow completed successfully")
}

// WebFeatureWorkflow simulates a local version of
// <REPO_ROOT>/workflows/web-features-repo/workflows.yaml.tftpl.
// The only difference is that the calls to the web feature consumer happen
// serially.
type WebFeatureWorkflow struct {
	webFeatureClient web_feature_consumer.ClientWithResponsesInterface
	githubOrg        string
	githubRepo       string
}

// newWebFeatureWorkflow creates a new WebFeatureWorkflow.
func newWebFeatureWorkflow(
	webFeatureClient web_feature_consumer.ClientWithResponsesInterface,
	githubOrg string,
	githubRepo string) WebFeatureWorkflow {

	return WebFeatureWorkflow{
		webFeatureClient: webFeatureClient,
		githubOrg:        githubOrg,
		githubRepo:       githubRepo,
	}
}

// Run executes the workflow.
func (w WebFeatureWorkflow) Run(ctx context.Context) error {
	webFeatureResp, err := w.webFeatureClient.PostV1WebFeaturesWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("web feature client call failed: %w", err)
	}
	if webFeatureResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to consume web features from repo. status %d", webFeatureResp.StatusCode())
	}

	return nil
}

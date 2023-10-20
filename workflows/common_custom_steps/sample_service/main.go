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

// Sample run-helloworld is a minimal Cloud Run service.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	log.Print("starting server...")
	http.HandleFunc("/", handler)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	// nolint:gosec // Will remove this sample service
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

type Response struct {
	Message string `json:"message"`
}

func handler(w http.ResponseWriter, _ *http.Request) {
	name := os.Getenv("NAME")
	if name == "" {
		name = "World"
	}
	resp := Response{
		Message: fmt.Sprintf("Hello %s!", name),
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(&resp)
	if err != nil {
		log.Println(err)
	}
}

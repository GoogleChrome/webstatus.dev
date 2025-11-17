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
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// nolint:lll // WONTFIX: useful comment
/*
createUserClaim creates the token that will be used to create a fake user in the auth emulator.

The user will be created so that it can fake a sign in with GitHub.

More details about the structure of the claims can be found below.

Currently the JWT ID looks like this:
eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJuYW1lIjoiQXdlc29tZSBVc2VyIDEiLCJlbWFpbCI6ImF3ZXNvbWUudXNlci4xQGV4YW1wbGUuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImF1dGhfdGltZSI6MTcyMjI2MjQ4MiwidXNlcl9pZCI6ImRuMWJlV3JOWFFGZm1XRUxjdUtkUmh1UlhnRlIiLCJmaXJlYmFzZSI6eyJpZGVudGl0aWVzIjp7ImVtYWlsIjpbImF3ZXNvbWUudXNlci4xQGV4YW1wbGUuY29tIl0sImdpdGh1Yi5jb20iOlsiNDEzMTU2OTA1NTg1NTE2MjI2MTgyNDM3NzIxOTEwOTI4MTcwMzM1NyJdfSwic2lnbl9pbl9wcm92aWRlciI6ImdpdGh1Yi5jb20ifSwiaWF0IjoxNzIyMjYyNDg1LCJleHAiOjE3MjIyNjYwODUsImF1ZCI6ImxvY2FsIiwiaXNzIjoiaHR0cHM6Ly9zZWN1cmV0b2tlbi5nb29nbGUuY29tL2xvY2FsIiwic3ViIjoiZG4xYmVXck5YUUZmbVdFTGN1S2RSaHVSWGdGUiJ9.
Using jwt.io, that decodes to:

	{
		"name": "Awesome User 1",
		"email": "awesome.user.1@example.com",
		"email_verified": true,
		"auth_time": 1722262482,
		"user_id": "dn1beWrNXQFfmWELcuKdRhuRXgFR",
		"firebase": {
			"identities": {
				"email": [
					"awesome.user.1@example.com"
				],
				"github.com": [
					"4131569055855162261824377219109281703357"
				]
			},
			"sign_in_provider": "github.com"
		},
		"iat": 1722262485,
		"exp": 1722266085,
		"aud": "local",
		"iss": "https://securetoken.google.com/local",
		"sub": "dn1beWrNXQFfmWELcuKdRhuRXgFR"
	}

	Inspiration for createUsers comes from:
	- https://github.com/firebase/firebase-tools/blob/1037f51ba2e07a1668b9ae334bec6ee457f02786/src/emulator/auth/idp.spec.ts#L37-L47
	- https://github.com/mikcsabee/auth-emulator-example/blob/4ec3c0f749dddda5afb92f5ee7435527a4774583/postman_pre-request_script.js#L31-L38
	- https://github.com/Dietarify/app-backend/blob/5c905a5c956a1542eebf28d0fbe1c7cab0ea80f4/README.md?plain=1#L70-L79
*/
func createUserClaim(user User, project string) jwt.MapClaims {
	issueTime := time.Now()
	claims := jwt.MapClaims{
		"name":           user.Name,
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"auth_time":      issueTime.Unix(),
		"user_id":        user.UserID,
		"firebase": map[string]interface{}{
			"identities": map[string]interface{}{
				"email": []string{
					user.Email,
				},
				"github.com": []string{
					fmt.Sprintf("%d", user.GitHubUserID),
				},
			},
			"sign_in_provider": "github.com",
		},
		"iat": issueTime.Unix(),
		"exp": issueTime.Add(1 * time.Hour).Unix(),
		"aud": project,
		"sub": user.UserID,
		"iss": "https://securetoken.google.com/" + project,
	}

	return claims
}

// getUsers contains the list of users to create.
func getUsers() []User {
	return []User{
		{
			Name:          "test user 1",
			Email:         "test.user.1@example.com",
			EmailVerified: true,
			UserID:        "1234567890",
			GitHubUserID:  1234567890,
		},
		{
			Name:          "test user 2",
			Email:         "test.user.2@example.com",
			EmailVerified: true,
			UserID:        "1234567891",
			GitHubUserID:  1234567891,
		},
		{
			Name:          "test user 3",
			Email:         "test.user.3@example.com",
			EmailVerified: true,
			UserID:        "1234567892",
			GitHubUserID:  1234567892,
		},
		// This user should have no data and should be used to replicate the experience of a newly logged in user.
		{
			Name:          "fresh user",
			Email:         "fresh.user@example.com",
			EmailVerified: true,
			UserID:        "1234567893",
			GitHubUserID:  1234567893,
		},
		{
			Name:          "chromium user",
			Email:         "chromium.user@example.com",
			EmailVerified: true,
			UserID:        "1234567894",
			GitHubUserID:  1234567894,
		},
		{
			Name:          "firefox user",
			Email:         "firefox.user@example.com",
			EmailVerified: true,
			UserID:        "1234567895",
			GitHubUserID:  1234567895,
		},
		{
			Name:          "webkit user",
			Email:         "webkit.user@example.com",
			EmailVerified: true,
			UserID:        "1234567896",
			GitHubUserID:  1234567896,
		},
	}
}

func createUsers(project string) {
	for _, user := range getUsers() {
		// Step 1. Build the token
		claim := createUserClaim(user, project)
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claim)
		if token == nil {
			slog.ErrorContext(context.TODO(), "missing token")
			os.Exit(1)
		}
		signedString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			slog.ErrorContext(context.TODO(), "unable to sign token", "error", err)
			os.Exit(1)
		}

		u := "http://localhost:9099/identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key=fake"

		var jsonData = []byte(fmt.Sprintf(`{
			"postBody": "providerId=github.com&id_token=%s",
			"requestUri": "http://localhost",
			"returnIdpCredential": true,
			"returnSecureToken": true
		}`, signedString))

		// Step 2. Send the request to create the account
		req, err := http.NewRequestWithContext(context.TODO(), http.MethodPost, u, bytes.NewBuffer(jsonData))
		if err != nil {
			slog.ErrorContext(context.TODO(), "unable to create request", "error", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.ErrorContext(context.TODO(), "unable to send request", "error", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			slog.ErrorContext(context.TODO(), "unexpected status code", "status code", resp.StatusCode)
			os.Exit(1)
		}

		slog.InfoContext(context.TODO(), "user created", "email", user.Email)
	}
}

type User struct {
	Name          string
	Email         string
	EmailVerified bool
	UserID        string
	// Use int64 to match the type used by GitHub.
	// For now, the GitHubUserID must match the UserID in the emulator
	// In real life, the UserID is the ID managed by GCIP. And GitHubUserID is the ID managed by GitHub.
	GitHubUserID int64
}

func main() {
	var (
		authProject = flag.String("project", "", "Google Cloud Identity Platform Project")
	)
	flag.Parse()
	createUsers(*authProject)
}

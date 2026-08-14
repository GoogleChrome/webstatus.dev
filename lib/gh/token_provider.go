// Copyright 2026 Google LLC
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

package gh

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v79/github"
)

var (
	// ErrEmptyInstallationID is returned when an installation ID is empty.
	ErrEmptyInstallationID = errors.New("installation id must not be empty")
)

// InstallationTokenProvider retrieves a valid installation access token for a GitHub App installation.
type InstallationTokenProvider interface {
	GetInstallationToken(ctx context.Context, installationID string) (string, error)
}

// TokenCacher caches installation access tokens.
type TokenCacher interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Cache(ctx context.Context, key string, in []byte) error
}

// TokenProvider manages RS256 JWT minting and cached GitHub App installation access tokens.
type TokenProvider struct {
	appID         string
	privateKeyPEM []byte
	httpClient    *http.Client
	baseURL       string
	cacher        TokenCacher
}

// TokenProviderOption configures a TokenProvider.
type TokenProviderOption func(*TokenProvider)

// WithTokenProviderHTTPClient sets a custom HTTP client for the TokenProvider.
func WithTokenProviderHTTPClient(client *http.Client) TokenProviderOption {
	return func(tp *TokenProvider) {
		if client != nil {
			tp.httpClient = client
		}
	}
}

// WithTokenProviderBaseURL sets a custom base URL for testing against WireMock or mock servers.
func WithTokenProviderBaseURL(baseURL string) TokenProviderOption {
	return func(tp *TokenProvider) {
		tp.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithTokenCacher sets a custom TokenCacher for caching installation tokens across workers.
func WithTokenCacher(cacher TokenCacher) TokenProviderOption {
	return func(tp *TokenProvider) {
		tp.cacher = cacher
	}
}

// NewTokenProvider returns a new TokenProvider configured with standard HTTP transport.
func NewTokenProvider(appID string, privateKeyPEM []byte, opts ...TokenProviderOption) *TokenProvider {
	tp := &TokenProvider{
		appID:         appID,
		privateKeyPEM: privateKeyPEM,
		httpClient: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       10 * time.Second,
		},
		baseURL: "https://api.github.com",
		cacher:  nil,
	}

	for _, opt := range opts {
		opt(tp)
	}

	return tp
}

// GetInstallationToken returns a valid GitHub App installation access token.
func (tp *TokenProvider) GetInstallationToken(ctx context.Context, installationID string) (string, error) {
	if installationID == "" {
		return "", ErrEmptyInstallationID
	}

	instID, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid installation id %q: %w", installationID, err)
	}

	cacheKey := "gh:inst_token:" + installationID
	if tp.cacher != nil {
		cachedBytes, cacheErr := tp.cacher.Get(ctx, cacheKey)
		if cacheErr == nil && len(cachedBytes) > 0 {
			return string(cachedBytes), nil
		}
	}

	jwtToken, mintErr := MintAppJWT(tp.appID, tp.privateKeyPEM)
	if mintErr != nil {
		return "", fmt.Errorf("failed to mint app jwt: %w", mintErr)
	}

	// Detach background context with 10s timeout to prevent cache poison on client ctx cancel
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	appClient := github.NewClient(tp.httpClient).WithAuthToken(jwtToken)
	if tp.baseURL != "https://api.github.com" && tp.baseURL != "" {
		baseURLWithSlash := strings.TrimRight(tp.baseURL, "/") + "/"
		customURL, parseErr := url.Parse(baseURLWithSlash)
		if parseErr == nil {
			appClient.BaseURL = customURL
		}
	}

	token, _, createErr := appClient.Apps.CreateInstallationToken(reqCtx, instID, nil)
	if createErr != nil {
		return "", fmt.Errorf("failed to create installation token: %w", createErr)
	}

	tokenStr := token.GetToken()
	if tokenStr == "" {
		return "", errors.New("empty installation access token received from github")
	}

	if tp.cacher != nil {
		_ = tp.cacher.Cache(ctx, cacheKey, []byte(tokenStr))
	}

	return tokenStr, nil
}

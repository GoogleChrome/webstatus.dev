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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	// ErrEmptyInstallationID is returned when an installation ID is empty.
	ErrEmptyInstallationID = errors.New("installation id must not be empty")
)

type cachedToken struct {
	token     string
	expiresAt time.Time
}

// TokenProvider manages RS256 JWT minting and cached GitHub App installation access tokens.
type TokenProvider struct {
	appID         string
	privateKeyPEM []byte
	httpClient    *http.Client
	baseURL       string
	cacheMu       sync.RWMutex
	cachedTokens  map[string]cachedToken
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
		baseURL:      "https://api.github.com",
		cacheMu:      sync.RWMutex{},
		cachedTokens: make(map[string]cachedToken),
	}

	for _, opt := range opts {
		opt(tp)
	}

	return tp
}

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetInstallationToken returns a valid GitHub App installation access token.
// Caches tokens in memory with a 5-minute expiry safety buffer.
func (tp *TokenProvider) GetInstallationToken(ctx context.Context, installationID string) (string, error) {
	if installationID == "" {
		return "", ErrEmptyInstallationID
	}

	// 1. Fast path: check memory cache under read lock
	tp.cacheMu.RLock()
	c, ok := tp.cachedTokens[installationID]
	if ok && time.Now().UTC().Before(c.expiresAt) {
		tp.cacheMu.RUnlock()

		return c.token, nil
	}
	tp.cacheMu.RUnlock()

	// 2. Slow path: fetch fresh token under write lock
	tp.cacheMu.Lock()
	defer tp.cacheMu.Unlock()

	// Double-check cache under write lock
	c, ok = tp.cachedTokens[installationID]
	if ok && time.Now().UTC().Before(c.expiresAt) {
		return c.token, nil
	}

	jwtToken, mintErr := MintAppJWT(tp.appID, tp.privateKeyPEM)
	if mintErr != nil {
		return "", fmt.Errorf("failed to mint app jwt: %w", mintErr)
	}

	// Detach background context with 10s timeout to prevent cache poison on client ctx cancel
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/app/installations/%s/access_tokens", tp.baseURL, installationID)
	req, reqErr := http.NewRequestWithContext(reqCtx, http.MethodPost, reqURL, nil)
	if reqErr != nil {
		return "", fmt.Errorf("failed to create access token request: %w", reqErr)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, doErr := tp.httpClient.Do(req)
	if doErr != nil {
		return "", fmt.Errorf("access token request failed: %w", doErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

		return "", fmt.Errorf("unexpected status %d from github access tokens API: %s", resp.StatusCode, string(body))
	}

	var tokenResp installationTokenResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResp); decodeErr != nil {
		return "", fmt.Errorf("failed to decode access token response: %w", decodeErr)
	}

	if tokenResp.Token == "" {
		return "", errors.New("empty installation access token received from github")
	}

	// 5-minute safety buffer before token true expiration
	cacheExpiry := tokenResp.ExpiresAt.UTC().Add(-5 * time.Minute)
	if cacheExpiry.Before(time.Now().UTC()) {
		cacheExpiry = tokenResp.ExpiresAt.UTC()
	}

	tp.cachedTokens[installationID] = cachedToken{
		token:     tokenResp.Token,
		expiresAt: cacheExpiry,
	}

	return tokenResp.Token, nil
}

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
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v79/github"
	"golang.org/x/sync/singleflight"
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

type cachedInstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TokenProvider manages RS256 JWT minting and cached GitHub App installation access tokens.
type TokenProvider struct {
	appID      string
	privateKey *rsa.PrivateKey
	httpClient *http.Client
	baseURL    string
	cacher     TokenCacher
	flight     singleflight.Group
}

// NewTokenProvider returns a new TokenProvider configured with the given cacher and standard HTTP transport.
// Validates appID and parses privateKeyPEM at construction time to fail fast on invalid credentials.
func NewTokenProvider(appID string, privateKeyPEM []byte, cacher TokenCacher) (*TokenProvider, error) {
	if appID == "" {
		return nil, ErrEmptyAppID
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPrivateKey, err)
	}

	return &TokenProvider{
		appID:      appID,
		privateKey: key,
		cacher:     cacher,
		httpClient: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       10 * time.Second,
		},
		baseURL: "https://api.github.com",
		flight:  singleflight.Group{},
	}, nil
}

func (tp *TokenProvider) getCachedToken(ctx context.Context, cacheKey string) (string, bool) {
	cachedBytes, cacheErr := tp.cacher.Get(ctx, cacheKey)
	if cacheErr != nil || len(cachedBytes) == 0 {
		return "", false
	}

	var cached cachedInstallationToken
	if err := json.Unmarshal(cachedBytes, &cached); err != nil {
		return "", false
	}

	if cached.Token != "" && time.Now().UTC().Add(time.Minute).Before(cached.ExpiresAt) {
		return cached.Token, true
	}

	return "", false
}

func (tp *TokenProvider) createInstallationToken(ctx context.Context, instID int64) (string, time.Time, error) {
	jwtToken, mintErr := mintAppJWTWithKey(tp.appID, tp.privateKey)
	if mintErr != nil {
		return "", time.Time{}, fmt.Errorf("failed to mint app jwt: %w", mintErr)
	}

	// Note: context.WithoutCancel is available in Go 1.21+ (repo uses Go 1.24).
	// Detach background context with 10s timeout to prevent cache poison on client ctx cancel.
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
		return "", time.Time{}, fmt.Errorf("failed to create installation token: %w", createErr)
	}

	tokenStr := token.GetToken()
	if tokenStr == "" {
		return "", time.Time{}, errors.New("empty installation access token received from github")
	}

	expiresAt := time.Now().UTC().Add(1 * time.Hour)
	if token.ExpiresAt != nil {
		expiresAt = token.GetExpiresAt().Time
	}

	return tokenStr, expiresAt, nil
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
	if tok, ok := tp.getCachedToken(ctx, cacheKey); ok {
		return tok, nil
	}

	res, err, _ := tp.flight.Do(installationID, func() (any, error) {
		if tok, ok := tp.getCachedToken(ctx, cacheKey); ok {
			return tok, nil
		}

		tokenStr, expiresAt, createErr := tp.createInstallationToken(ctx, instID)
		if createErr != nil {
			return "", createErr
		}

		cachedJSON, marshalErr := json.Marshal(cachedInstallationToken{
			Token:     tokenStr,
			ExpiresAt: expiresAt,
		})
		if marshalErr != nil {
			slog.WarnContext(ctx, "failed to marshal installation token for cache",
				"error", marshalErr,
				"installation_id", installationID)
		} else if cacheErr := tp.cacher.Cache(ctx, cacheKey, cachedJSON); cacheErr != nil {
			slog.WarnContext(ctx, "failed to cache installation token",
				"error", cacheErr,
				"installation_id", installationID)
		}

		return tokenStr, nil
	})
	if err != nil {
		return "", err
	}

	// Note: singleflight.Group currently returns (any, error).
	// TODO: Replace type assertion once golang.org/x/sync/singleflight adds generic support.
	tokenStr, ok := res.(string)
	if !ok {
		return "", errors.New("unexpected token type from singleflight")
	}

	return tokenStr, nil
}

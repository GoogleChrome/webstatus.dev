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

package valkeycache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/cenkalti/backoff/v5"
	"github.com/valkey-io/valkey-go"
)

// ValkeyDataCache is a cache that relies on valkey.
// It uses generics so that users of it can uses any type of data they want.
// The key K must be of type comparable. More infomration here: https://go.dev/blog/comparable
// The value V can be any type.
type ValkeyDataCache[K comparable, V []byte] struct {
	keyPrefix string
	client    valkey.Client
	ttl       time.Duration
}

// NewValkeyDataCache creates a new ValkeyDataCache instance.
func NewValkeyDataCache[K comparable, V []byte](
	keyPrefix string,
	host string,
	port string, // Will likely come from the environment variable as a string
	ttl time.Duration) (*ValkeyDataCache[K, V], error) {

	addr := fmt.Sprintf("%s:%s", host, port)
	operation := func() (valkey.Client, error) {
		// nolint: exhaustruct // No need to use every option of 3rd party struct.
		return valkey.NewClient(valkey.ClientOption{
			InitAddress: []string{addr},
		})
	}

	c, err := backoff.Retry(context.TODO(), operation,
		backoff.WithBackOff(backoff.NewExponentialBackOff()),
		// Should be less than the total time in the startup probe for the backend container in
		// infra/backend/service.tf
		backoff.WithMaxElapsedTime(25*time.Second),
	)
	if err != nil {
		return nil, err
	}

	return &ValkeyDataCache[K, V]{
		keyPrefix: keyPrefix,
		client:    c,
		ttl:       ttl,
	}, nil
}

func (c *ValkeyDataCache[K, V]) cacheKey(key K) string {
	return fmt.Sprintf("%s-%v", c.keyPrefix, key)
}

// Cache stores a value in the cache.
func (c *ValkeyDataCache[K, V]) Cache(
	ctx context.Context,
	key K,
	in V,
	options ...cachetypes.CacheOption,
) error {
	// Build default config for cache operation
	cacheCfg := cachetypes.NewCacheConfig(c.ttl)

	// Apply options to config
	for _, opt := range options {
		opt(cacheCfg)
	}

	err := c.client.Do(ctx, c.client.B().Set().Key(c.cacheKey(key)).
		Value(valkey.BinaryString(in)).Ex(cacheCfg.GetTTL()).Build()).Error()
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a value from the cache.
// It returns cachetypes.ErrCachedDataNotFound if it does not exist.
// nolint: ireturn // V is not a interface always. Can ignore this.
func (c *ValkeyDataCache[K, V]) Get(
	ctx context.Context,
	key K,
) (V, error) {
	defaultValue := *new(V)
	msg, err := c.client.Do(ctx, c.client.B().Get().Key(c.cacheKey(key)).Build()).ToMessage()
	if errors.Is(err, valkey.Nil) {
		return defaultValue, cachetypes.ErrCachedDataNotFound
	} else if err != nil {
		// All other errors
		return defaultValue, err
	}

	return msg.AsBytes()
}

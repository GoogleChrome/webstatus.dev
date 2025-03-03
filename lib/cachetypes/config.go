// Copyright 2025 Google LLC
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

package cachetypes

import "time"

// CacheConfig contains the configuration when caching.
type CacheConfig struct {
	ttl time.Duration
}

func (c CacheConfig) GetTTL() time.Duration {
	return c.ttl
}

// NewCacheConfig creates a new CacheConfig instance.
func NewCacheConfig(ttl time.Duration) *CacheConfig {
	return &CacheConfig{
		ttl: ttl,
	}
}

// CacheOption is a function that configures the cache operation.
type CacheOption func(*CacheConfig)

// WithTTL sets the TTL for the cache.
func WithTTL(ttl time.Duration) CacheOption {
	return func(c *CacheConfig) {
		c.ttl = ttl
	}
}

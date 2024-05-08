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

package rediscache

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/gomodule/redigo/redis"
)

// RedisCache is a cache that relies on redis.
// It uses generics so that users of it can uses any type of data they want.
// The key K must be of type comparable. More infomration here: https://go.dev/blog/comparable
// The value V can be any type.
type RedisDataCache[K comparable, V any] struct {
	keyPrefix  string
	redisPool  *redis.Pool
	ttlSeconds int64
}

// NewRedisDataCache creates a new RedisDataCache instance.
func NewRedisDataCache[K comparable, V any](
	keyPrefix string,
	host string,
	port string, // Will likely come from the environment variable as a string
	ttl time.Duration,
	maxConnections int) (*RedisDataCache[K, V], error) {

	redisAddr := fmt.Sprintf("%s:%s", host, port)
	// nolint: exhaustruct // No need to use every option of 3rd party struct.
	redisPool := &redis.Pool{
		MaxIdle: maxConnections,
		Dial:    func() (redis.Conn, error) { return redis.Dial("tcp", redisAddr) },
	}

	return &RedisDataCache[K, V]{
		keyPrefix:  keyPrefix,
		redisPool:  redisPool,
		ttlSeconds: int64(ttl.Seconds()),
	}, nil
}

func (c *RedisDataCache[K, V]) cacheKey(key K) string {
	return fmt.Sprintf("%s-%v", c.keyPrefix, key)
}

// Cache stores a value in the cache.
func (c *RedisDataCache[K, V]) Cache(
	ctx context.Context,
	key K,
	in V,
) error {
	conn, err := c.redisPool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Do("SET", c.cacheKey(key), in, "EX", c.ttlSeconds)
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a value from the cache.
// It returns cachetypes.ErrCachedDataNotFound if it does not exist.
// nolint: ireturn // V is not a interface always. Can ignore this.
func (c *RedisDataCache[K, V]) Get(
	ctx context.Context,
	key K,
) (V, error) {
	conn, err := c.redisPool.GetContext(ctx)
	if err != nil {
		return *new(V), err
	}
	defer conn.Close()
	rawResult, err := conn.Do("GET", c.cacheKey(key))
	if err != nil {
		return *new(V), err
	} else if rawResult == nil {
		return *new(V), cachetypes.ErrCachedDataNotFound
	}

	result, ok := rawResult.(V)
	if !ok {
		return *new(V), cachetypes.ErrInvalidValueType
	}

	return result, nil
}

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
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/GoogleChrome/webstatus.dev/lib/cachetypes"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func getDefatulTTL() time.Duration { return time.Duration(2 * time.Second) }

// nolint: exhaustruct // No need to use every option of 3rd party struct.
func getTestRedis(t testing.TB) *RedisDataCache[string, []byte] {
	ctx := context.Background()
	repoRoot, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Error(err)
	}
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Dockerfile: filepath.Join(".dev", "redis", "Dockerfile"),
			Context:    repoRoot,
		},
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
		Name:         "webstatus-dev-test-redis",
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}

	mappedPort, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Error(err)
	}

	cache, err := NewRedisDataCache[string, []byte](
		"testPrefix",
		"localhost",
		mappedPort.Port(),
		getDefatulTTL(),
		10,
	)
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		cache.redisPool.Close()
	})

	return cache
}

func TestRedisDataCache(t *testing.T) {
	cache := getTestRedis(t)
	ctx := context.Background()

	testKey1 := "test-key-1"
	testValue1 := []byte("test-value")

	t.Run("cache miss", func(t *testing.T) {
		result, err := cache.Get(ctx, testKey1)
		if !errors.Is(err, cachetypes.ErrCachedDataNotFound) {
			t.Errorf("invalid error %v", err)
		}
		if result != nil {
			t.Error("expected null result")
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		// Store result
		err := cache.Cache(ctx, testKey1, testValue1)
		if !errors.Is(err, nil) {
			t.Errorf("invalid error storing value %v", err)
		}

		// Get result.
		result, err := cache.Get(ctx, testKey1)
		if !errors.Is(err, nil) {
			t.Errorf("invalid error getting value %v", err)
		}
		if !reflect.DeepEqual(result, testValue1) {
			t.Error("expected result")
		}

		// Wait for TTL
		time.Sleep(getDefatulTTL() * 2)
		result, err = cache.Get(ctx, testKey1)
		if !errors.Is(err, cachetypes.ErrCachedDataNotFound) {
			t.Errorf("invalid error getting expired result %v", err)
		}
		if result != nil {
			t.Error("expected null result")
		}

	})

}

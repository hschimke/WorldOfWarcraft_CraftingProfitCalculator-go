package cache_provider

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheProvider struct {
	redisClient *redis.Client
	ctx         context.Context
}

func NewCacheProvider(ctx context.Context, uri string) *CacheProvider {
	redis_options, err := redis.ParseURL(uri)
	if err != nil {
		panic("redis cannot be contacted")
	}
	return &CacheProvider{
		redisClient: redis.NewClient(redis_options),
		ctx:         ctx,
	}
}

// Create a Redis key given a namespace and key
func getRedisKey(namespace string, key string) string {
	return namespace + ":->" + key
}

// Fetch an object from the cache, or fail
func CacheGet[T any](cache *CacheProvider, namespace string, key string, target *T) error {
	data, getErr := cache.redisClient.Get(cache.ctx, getRedisKey(namespace, key)).Bytes()
	if getErr != nil {
		return getErr
	}
	convertErr := json.Unmarshal(data, &target)
	if convertErr != nil {
		return convertErr
	}
	return nil
}

// Set or replace an item in the cache, or fail.
func CacheSet[T any](cache *CacheProvider, namespace string, key string, data T, expiration_period time.Duration) error {
	json_data, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	setErr := cache.redisClient.Set(cache.ctx, getRedisKey(namespace, key), json_data, expiration_period).Err()
	if setErr != nil {
		return setErr
	}
	return nil
}

// Check if a given key exists in a given namespace
func CacheCheck(cache *CacheProvider, namespace string, key string) (bool, error) {
	//return false, nil
	fnd, err := cache.redisClient.Exists(cache.ctx, getRedisKey(namespace, key)).Result()
	if err != nil {
		return false, err
	}
	return fnd == 1, nil
}

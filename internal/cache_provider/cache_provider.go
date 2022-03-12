package cache_provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
)

var (
	redisClient *redis.Client
	ctx         context.Context
)

// Create a Redis key given a namespace and key
func getRedisKey(namespace string, key string) string {
	return fmt.Sprint(namespace, ":->", key)
}

// Fetch an object from the cache, or fail
func CacheGet[T any](namespace string, key string, target *T) error {
	data, getErr := redisClient.Get(ctx, getRedisKey(namespace, key)).Result()
	if getErr != nil {
		return getErr
	}
	convertErr := json.Unmarshal([]byte(data), &target)
	if convertErr != nil {
		return convertErr
	}
	return nil
}

// Set or replace an item in the cache, or fail.
func CacheSet[T any](namespace string, key string, data T, expiration_period time.Duration) error {
	json_data, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	setErr := redisClient.Set(ctx, getRedisKey(namespace, key), json_data, expiration_period).Err()
	if setErr != nil {
		return setErr
	}
	return nil
}

// Check if a given key exists in a given namespace
func CacheCheck(namespace string, key string) (bool, error) {
	//return false, nil
	fnd, err := redisClient.Exists(ctx, getRedisKey(namespace, key)).Result()
	if err != nil {
		return false, err
	}
	return fnd == 1, nil
}

func init() {
	uri := environment_variables.REDIS_URL

	ctx = context.Background()

	redis_options, err := redis.ParseURL(uri)
	if err != nil {
		panic("redis cannot be contacted")
	}
	redisClient = redis.NewClient(redis_options)
}

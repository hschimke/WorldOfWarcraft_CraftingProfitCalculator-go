package cache_provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/environment_variables"
)

var redisClient *redis.Client
var ctx context.Context

func getRedisKey(namespace string, key string) string {
	return fmt.Sprint(namespace, ":->", key)
}

func CacheGet(namespace string, key string, target interface{}) error {
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
func CacheSet(namespace string, key string, data interface{}, expiration_period time.Duration) error {
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
func CacheCheck(namespace string, key string) (bool, error) {
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

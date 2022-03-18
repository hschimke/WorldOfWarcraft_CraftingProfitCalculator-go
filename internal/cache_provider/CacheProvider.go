package cache_provider

import (
	"context"

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

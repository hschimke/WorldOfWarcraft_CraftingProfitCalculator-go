package routes

import (
	"context"

	"github.com/go-redis/redis/v8"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
)

type CPCRoutes struct {
	auctionHouseServer *auction_history.AuctionHistoryServer
	redisClient        *redis.Client
	helper             *blizzard_api_helpers.BlizzardApiHelper
	cache              *cache_provider.CacheProvider
	staticSources      static_sources.StaticSources
	Logger             *cpclog.CpCLog
	ctx                context.Context
}

func NewCPCRoutes(ctx context.Context, connectionString, redisUri string, helper *blizzard_api_helpers.BlizzardApiHelper, cache *cache_provider.CacheProvider, logger *cpclog.CpCLog) *CPCRoutes {
	redis_options, err := redis.ParseURL(redisUri)
	if err != nil {
		panic("redis cannot be contacted")
	}
	return &CPCRoutes{
		auctionHouseServer: auction_history.NewAuctionHistoryServer(ctx, connectionString, helper, logger),
		redisClient:        redis.NewClient(redis_options),
		helper:             helper,
		cache:              cache,
		Logger:             logger,
		ctx:                ctx,
	}
}

func (r *CPCRoutes) Shutdown() {
	r.auctionHouseServer.Shutdown()
}

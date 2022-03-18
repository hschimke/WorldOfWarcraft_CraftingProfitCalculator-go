package blizzard_api_helpers

import (
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
)

type BlizzardApiHelper struct {
	api    *blizzard_api_call.BlizzardApiProvider
	cache  *cache_provider.CacheProvider
	logger *cpclog.CpCLog
}

func NewBlizzardApiHelper(cache *cache_provider.CacheProvider, logger *cpclog.CpCLog, api *blizzard_api_call.BlizzardApiProvider) *BlizzardApiHelper {
	return &BlizzardApiHelper{
		api:    api,
		cache:  cache,
		logger: logger,
	}
}

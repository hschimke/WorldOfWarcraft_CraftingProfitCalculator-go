package blizzard_api_helpers

import (
	"fmt"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/redis_cache_provider"
)

func GetItemDetails(item_id globalTypes.ItemID, region globalTypes.RegionCode) (BlizzardApi.Item, error) {
	var key = fmt.Sprint(item_id)

	if found, err := redis_cache_provider.CacheCheck(ITEM_DATA_CACHE, key); err == nil && found {
		item := BlizzardApi.Item{}
		fndErr := redis_cache_provider.CacheGet(ITEM_DATA_CACHE, key, &item)
		return item, fndErr
	}

	var profession_item_detail_uri string = fmt.Sprintf("/data/wow/item/%s", item_id)
	//categories[array].recipes[array].name categories[array].recipes[array].id
	result := BlizzardApi.Item{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		getNamespace(static_ns, region),
		locale_us,
	}, profession_item_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Item{}, fetchErr
	}
	redis_cache_provider.CacheSet(ITEM_DATA_CACHE, key, &result, redis_cache_provider.GetStaticTimeWithShift())
	return result, nil

}

func GetBlizProfessionsList(region globalTypes.RegionCode) (BlizzardApi.ProfessionsIndex, error) {

	key := region
	const profession_list_uri string = "/data/wow/profession/index" // professions.name / professions.id

	if found, err := redis_cache_provider.CacheCheck(PROFESSION_LIST_CACHE, key); err == nil && found {
		item := BlizzardApi.ProfessionsIndex{}
		fndErr := redis_cache_provider.CacheGet(PROFESSION_LIST_CACHE, key, &item)
		return item, fndErr
	}

	result := BlizzardApi.ProfessionsIndex{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		getNamespace(static_ns, region),
		locale_us,
	}, profession_list_uri, result)
	if fetchErr != nil {
		return BlizzardApi.ProfessionsIndex{}, fetchErr
	}
	redis_cache_provider.CacheSet(PROFESSION_LIST_CACHE, key, &result, redis_cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetBlizProfessionDetail(profession_id uint, region globalTypes.RegionCode) (BlizzardApi.Profession, error) {
	key := fmt.Sprintf("%s::%s", region, profession_id)

	if found, err := redis_cache_provider.CacheCheck(PROFESSION_DETAIL_CACHE, key); err == nil && found {
		item := BlizzardApi.Profession{}
		fndErr := redis_cache_provider.CacheGet(PROFESSION_DETAIL_CACHE, key, &item)
		return item, fndErr
	}

	profession_detail_uri := fmt.Sprintf("/data/wow/profession/%s", profession_id)
	result := BlizzardApi.Profession{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		getNamespace(static_ns, region),
		locale_us,
	}, profession_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Profession{}, fetchErr
	}
	redis_cache_provider.CacheSet(PROFESSION_DETAIL_CACHE, key, &result, redis_cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetBlizConnectedRealmDetail(connected_realm_id globalTypes.ConnectedRealmID, region globalTypes.RegionCode) (BlizzardApi.ConnectedRealm, error) {
	key := fmt.Sprintf("%s::%s", region, connected_realm_id)

	if found, err := redis_cache_provider.CacheCheck(COMPOSITE_REALM_NAME_CACHE, key); err == nil && found {
		item := BlizzardApi.ConnectedRealm{}
		fndErr := redis_cache_provider.CacheGet(COMPOSITE_REALM_NAME_CACHE, key, &item)
		return item, fndErr
	}

	connected_realm_detail_uri := fmt.Sprintf("/data/wow/connected-realm/%s", connected_realm_id)
	result := BlizzardApi.ConnectedRealm{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		getNamespace(dynamic_ns, region),
		locale_us,
	}, connected_realm_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.ConnectedRealm{}, fetchErr
	}
	redis_cache_provider.CacheSet(COMPOSITE_REALM_NAME_CACHE, key, &result, redis_cache_provider.GetDynamicTimeWithShift())
	return result, nil
}

func GetBlizSkillTierDetail(profession_id uint, skillTier_id uint, region globalTypes.RegionCode) (BlizzardApi.ProfessionSkillTier, error) {
	key := fmt.Sprintf("%s::%s::%s", region, profession_id, skillTier_id)

	if found, err := redis_cache_provider.CacheCheck(PROFESSION_SKILL_TIER_DETAILS_CACHE, key); err == nil && found {
		item := BlizzardApi.ProfessionSkillTier{}
		fndErr := redis_cache_provider.CacheGet(PROFESSION_SKILL_TIER_DETAILS_CACHE, key, &item)
		return item, fndErr
	}

	profession_skill_tier_detail_uri := fmt.Sprintf("/data/wow/profession/%s/skill-tier/%s", profession_id, skillTier_id)
	result := BlizzardApi.ProfessionSkillTier{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		getNamespace(static_ns, region),
		locale_us,
	}, profession_skill_tier_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.ProfessionSkillTier{}, fetchErr
	}
	redis_cache_provider.CacheSet(PROFESSION_SKILL_TIER_DETAILS_CACHE, key, &result, redis_cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetBlizRecipeDetail(recipe_id uint, region globalTypes.RegionCode) (BlizzardApi.Recipe, error) {
	key := fmt.Sprintf("%s::%s", region, region, recipe_id)

	if found, err := redis_cache_provider.CacheCheck(PROFESSION_RECIPE_DETAIL_CACHE, key); err == nil && found {
		item := BlizzardApi.Recipe{}
		fndErr := redis_cache_provider.CacheGet(PROFESSION_RECIPE_DETAIL_CACHE, key, &item)
		return item, fndErr
	}

	profession_recipe_uri := fmt.Sprintf("/data/wow/recipe/%s", recipe_id)
	result := BlizzardApi.Recipe{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		getNamespace(static_ns, region),
		locale_us,
	}, profession_recipe_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Recipe{}, fetchErr
	}
	redis_cache_provider.CacheSet(PROFESSION_RECIPE_DETAIL_CACHE, key, &result, redis_cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetAuctionHouse(server_id globalTypes.ConnectedRealmID, server_region globalTypes.RegionCode) (BlizzardApi.Auctions, error) {
	key := fmt.Sprintf("%s", server_id)

	if found, err := redis_cache_provider.CacheCheck(AUCTION_DATA_CACHE, key); err == nil && found {
		item := BlizzardApi.Auctions{}
		fndErr := redis_cache_provider.CacheGet(AUCTION_DATA_CACHE, key, &item)
		return item, fndErr
	}

	auction_house_fetch_uri := fmt.Sprintf("/data/wow/connected-realm/%s/auctions", server_id)
	result := BlizzardApi.Auctions{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(server_region, basicDataPackage{
		getNamespace(dynamic_ns, server_region),
		"",
	}, auction_house_fetch_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Auctions{}, fetchErr
	}
	redis_cache_provider.CacheSet(AUCTION_DATA_CACHE, key, &result, time.Duration(time.Hour*1))
	return result, nil
}

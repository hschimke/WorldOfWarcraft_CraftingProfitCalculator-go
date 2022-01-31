package blizzard_api_helpers

import (
	"fmt"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

func GetItemDetails(item_id globalTypes.ItemID, region globalTypes.RegionCode) (BlizzardApi.Item, error) {
	var key = fmt.Sprint(item_id)

	if found, err := cache_provider.CacheCheck(ITEM_DATA_CACHE, key); err == nil && found {
		item := BlizzardApi.Item{}
		fndErr := cache_provider.CacheGet(ITEM_DATA_CACHE, key, &item)
		return item, fndErr
	}

	var profession_item_detail_uri string = fmt.Sprintf("/data/wow/item/%d", item_id)
	//categories[array].recipes[array].name categories[array].recipes[array].id
	result := BlizzardApi.Item{}

	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		"namespace": getNamespace(static_ns, region),
		"locale":    locale_us,
	}, profession_item_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Item{}, fetchErr
	}
	cache_provider.CacheSet(ITEM_DATA_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil

}

func GetBlizProfessionsList(region globalTypes.RegionCode) (BlizzardApi.ProfessionsIndex, error) {

	key := region
	const profession_list_uri string = "/data/wow/profession/index" // professions.name / professions.id

	if found, err := cache_provider.CacheCheck(PROFESSION_LIST_CACHE, key); err == nil && found {
		item := BlizzardApi.ProfessionsIndex{}
		fndErr := cache_provider.CacheGet(PROFESSION_LIST_CACHE, key, &item)
		return item, fndErr
	}

	result := BlizzardApi.ProfessionsIndex{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		"namespace": getNamespace(static_ns, region),
		"locale":    locale_us,
	}, profession_list_uri, result)
	if fetchErr != nil {
		return BlizzardApi.ProfessionsIndex{}, fetchErr
	}
	cache_provider.CacheSet(PROFESSION_LIST_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetBlizProfessionDetail(profession_id uint, region globalTypes.RegionCode) (BlizzardApi.Profession, error) {
	key := fmt.Sprintf("%s::%d", region, profession_id)

	if found, err := cache_provider.CacheCheck(PROFESSION_DETAIL_CACHE, key); err == nil && found {
		item := BlizzardApi.Profession{}
		fndErr := cache_provider.CacheGet(PROFESSION_DETAIL_CACHE, key, &item)
		return item, fndErr
	}

	profession_detail_uri := fmt.Sprintf("/data/wow/profession/%d", profession_id)
	result := BlizzardApi.Profession{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		"namespace": getNamespace(static_ns, region),
		"locale":    locale_us,
	}, profession_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Profession{}, fetchErr
	}
	cache_provider.CacheSet(PROFESSION_DETAIL_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetBlizConnectedRealmDetail(connected_realm_id globalTypes.ConnectedRealmID, region globalTypes.RegionCode) (BlizzardApi.ConnectedRealm, error) {
	key := fmt.Sprintf("%s::%d", region, connected_realm_id)

	if found, err := cache_provider.CacheCheck(COMPOSITE_REALM_NAME_CACHE, key); err == nil && found {
		item := BlizzardApi.ConnectedRealm{}
		fndErr := cache_provider.CacheGet(COMPOSITE_REALM_NAME_CACHE, key, &item)
		return item, fndErr
	}

	connected_realm_detail_uri := fmt.Sprintf("/data/wow/connected-realm/%d", connected_realm_id)
	result := BlizzardApi.ConnectedRealm{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		"namespace": getNamespace(dynamic_ns, region),
		"locale":    locale_us,
	}, connected_realm_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.ConnectedRealm{}, fetchErr
	}
	cache_provider.CacheSet(COMPOSITE_REALM_NAME_CACHE, key, &result, cache_provider.GetDynamicTimeWithShift())
	return result, nil
}

func GetBlizSkillTierDetail(profession_id uint, skillTier_id uint, region globalTypes.RegionCode) (BlizzardApi.ProfessionSkillTier, error) {
	key := fmt.Sprintf("%s::%d::%d", region, profession_id, skillTier_id)

	if found, err := cache_provider.CacheCheck(PROFESSION_SKILL_TIER_DETAILS_CACHE, key); err == nil && found {
		item := BlizzardApi.ProfessionSkillTier{}
		fndErr := cache_provider.CacheGet(PROFESSION_SKILL_TIER_DETAILS_CACHE, key, &item)
		return item, fndErr
	}

	profession_skill_tier_detail_uri := fmt.Sprintf("/data/wow/profession/%d/skill-tier/%d", profession_id, skillTier_id)
	result := BlizzardApi.ProfessionSkillTier{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		"namespace": getNamespace(static_ns, region),
		"locale":    locale_us,
	}, profession_skill_tier_detail_uri, result)
	if fetchErr != nil {
		return BlizzardApi.ProfessionSkillTier{}, fetchErr
	}
	cache_provider.CacheSet(PROFESSION_SKILL_TIER_DETAILS_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetBlizRecipeDetail(recipe_id uint, region globalTypes.RegionCode) (BlizzardApi.Recipe, error) {
	key := fmt.Sprintf("%s::%d", region, recipe_id)

	if found, err := cache_provider.CacheCheck(PROFESSION_RECIPE_DETAIL_CACHE, key); err == nil && found {
		item := BlizzardApi.Recipe{}
		fndErr := cache_provider.CacheGet(PROFESSION_RECIPE_DETAIL_CACHE, key, &item)
		return item, fndErr
	}

	profession_recipe_uri := fmt.Sprintf("/data/wow/recipe/%d", recipe_id)
	result := BlizzardApi.Recipe{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(region, basicDataPackage{
		"namespace": getNamespace(static_ns, region),
		"locale":    locale_us,
	}, profession_recipe_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Recipe{}, fetchErr
	}
	cache_provider.CacheSet(PROFESSION_RECIPE_DETAIL_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

func GetAuctionHouse(server_id globalTypes.ConnectedRealmID, server_region globalTypes.RegionCode) (BlizzardApi.Auctions, error) {
	key := fmt.Sprint(server_id)

	if found, err := cache_provider.CacheCheck(AUCTION_DATA_CACHE, key); err == nil && found {
		item := BlizzardApi.Auctions{}
		fndErr := cache_provider.CacheGet(AUCTION_DATA_CACHE, key, &item)
		return item, fndErr
	}

	auction_house_fetch_uri := fmt.Sprintf("/data/wow/connected-realm/%d/auctions", server_id)
	result := BlizzardApi.Auctions{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(server_region, basicDataPackage{
		"namespace": getNamespace(dynamic_ns, server_region),
	}, auction_house_fetch_uri, result)
	if fetchErr != nil {
		return BlizzardApi.Auctions{}, fetchErr
	}
	cache_provider.CacheSet(AUCTION_DATA_CACHE, key, &result, time.Duration(time.Hour*1))
	return result, nil
}

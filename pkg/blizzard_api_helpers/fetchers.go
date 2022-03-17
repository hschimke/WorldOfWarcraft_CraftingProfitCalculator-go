package blizzard_api_helpers

import (
	"fmt"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

const (
	getItemDetailsUri              string = "/data/wow/item/%d"
	profession_list_uri            string = "/data/wow/profession/index" // professions.name / professions.id
	getBlizProfessionDetailUri     string = "/data/wow/profession/%d"
	getBlizConnectedRealmDetailUri string = "/data/wow/connected-realm/%d"
	getBlizSkillTierDetailUri      string = "/data/wow/profession/%d/skill-tier/%d"
	getBlizRecipeDetailUri         string = "/data/wow/recipe/%d"
	getAuctionHouseUri             string = "/data/wow/connected-realm/%d/auctions"
)

// Fetch item details from Blizzard API
func (helper *BlizzardApiHelper) GetItemDetails(item_id globalTypes.ItemID, region globalTypes.RegionCode) (BlizzardApi.Item, error) {
	var key = fmt.Sprint(item_id)

	if found, err := cache_provider.CacheCheck(helper.cache, ITEM_DATA_CACHE, key); err == nil && found {
		item := BlizzardApi.Item{}
		fndErr := cache_provider.CacheGet(helper.cache, ITEM_DATA_CACHE, key, &item)
		return item, fndErr
	}

	var profession_item_detail_uri string = fmt.Sprintf(getItemDetailsUri, item_id)
	//categories[array].recipes[array].name categories[array].recipes[array].id
	result := BlizzardApi.Item{}

	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(helper.api, region, basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}, profession_item_detail_uri, getNamespace(static_ns, region), &result)
	if fetchErr != nil {
		return BlizzardApi.Item{}, fetchErr
	}
	cache_provider.CacheSet(helper.cache, ITEM_DATA_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil

}

// Fetch a list of all professions from Blizzard API
func (helper *BlizzardApiHelper) GetBlizProfessionsList(region globalTypes.RegionCode) (BlizzardApi.ProfessionsIndex, error) {

	key := region

	if found, err := cache_provider.CacheCheck(helper.cache, PROFESSION_LIST_CACHE, key); err == nil && found {
		item := BlizzardApi.ProfessionsIndex{}
		fndErr := cache_provider.CacheGet(helper.cache, PROFESSION_LIST_CACHE, key, &item)
		return item, fndErr
	}

	result := BlizzardApi.ProfessionsIndex{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(helper.api, region, basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}, profession_list_uri, getNamespace(static_ns, region), &result)
	if fetchErr != nil {
		return BlizzardApi.ProfessionsIndex{}, fetchErr
	}
	cache_provider.CacheSet(helper.cache, PROFESSION_LIST_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

// Fetch Profession details from Blizzard API
func (helper *BlizzardApiHelper) GetBlizProfessionDetail(profession_id uint, region globalTypes.RegionCode) (BlizzardApi.Profession, error) {
	key := fmt.Sprintf("%s::%d", region, profession_id)

	if found, err := cache_provider.CacheCheck(helper.cache, PROFESSION_DETAIL_CACHE, key); err == nil && found {
		item := BlizzardApi.Profession{}
		fndErr := cache_provider.CacheGet(helper.cache, PROFESSION_DETAIL_CACHE, key, &item)
		return item, fndErr
	}

	profession_detail_uri := fmt.Sprintf(getBlizProfessionDetailUri, profession_id)
	result := BlizzardApi.Profession{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(helper.api, region, basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}, profession_detail_uri, getNamespace(static_ns, region), &result)
	if fetchErr != nil {
		return BlizzardApi.Profession{}, fetchErr
	}
	cache_provider.CacheSet(helper.cache, PROFESSION_DETAIL_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

// Get details about a connected realm from Blizzard API
func (helper *BlizzardApiHelper) GetBlizConnectedRealmDetail(connected_realm_id globalTypes.ConnectedRealmID, region globalTypes.RegionCode) (BlizzardApi.ConnectedRealm, error) {
	key := fmt.Sprintf("%s::%d", region, connected_realm_id)

	if found, err := cache_provider.CacheCheck(helper.cache, COMPOSITE_REALM_NAME_CACHE, key); err == nil && found {
		item := BlizzardApi.ConnectedRealm{}
		fndErr := cache_provider.CacheGet(helper.cache, COMPOSITE_REALM_NAME_CACHE, key, &item)
		return item, fndErr
	}

	connected_realm_detail_uri := fmt.Sprintf(getBlizConnectedRealmDetailUri, connected_realm_id)
	result := BlizzardApi.ConnectedRealm{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(helper.api, region, basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}, connected_realm_detail_uri, getNamespace(dynamic_ns, region), &result)
	if fetchErr != nil {
		return BlizzardApi.ConnectedRealm{}, fetchErr
	}
	cache_provider.CacheSet(helper.cache, COMPOSITE_REALM_NAME_CACHE, key, &result, cache_provider.GetDynamicTimeWithShift())
	return result, nil
}

// Fetch an individual crafting skill tier from Blizzard API
func (helper *BlizzardApiHelper) GetBlizSkillTierDetail(profession_id uint, skillTier_id uint, region globalTypes.RegionCode) (BlizzardApi.ProfessionSkillTier, error) {
	key := fmt.Sprintf("%s::%d::%d", region, profession_id, skillTier_id)

	if found, err := cache_provider.CacheCheck(helper.cache, PROFESSION_SKILL_TIER_DETAILS_CACHE, key); err == nil && found {
		item := BlizzardApi.ProfessionSkillTier{}
		fndErr := cache_provider.CacheGet(helper.cache, PROFESSION_SKILL_TIER_DETAILS_CACHE, key, &item)
		return item, fndErr
	}

	profession_skill_tier_detail_uri := fmt.Sprintf(getBlizSkillTierDetailUri, profession_id, skillTier_id)
	result := BlizzardApi.ProfessionSkillTier{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(helper.api, region, basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}, profession_skill_tier_detail_uri, getNamespace(static_ns, region), &result)
	if fetchErr != nil {
		return BlizzardApi.ProfessionSkillTier{}, fetchErr
	}
	cache_provider.CacheSet(helper.cache, PROFESSION_SKILL_TIER_DETAILS_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

// Fetch a recipe detail from Blizzard API
func (helper *BlizzardApiHelper) GetBlizRecipeDetail(recipe_id uint, region globalTypes.RegionCode) (BlizzardApi.Recipe, error) {
	key := fmt.Sprintf("%s::%d", region, recipe_id)

	if found, err := cache_provider.CacheCheck(helper.cache, PROFESSION_RECIPE_DETAIL_CACHE, key); err == nil && found {
		item := BlizzardApi.Recipe{}
		fndErr := cache_provider.CacheGet(helper.cache, PROFESSION_RECIPE_DETAIL_CACHE, key, &item)
		return item, fndErr
	}

	profession_recipe_uri := fmt.Sprintf(getBlizRecipeDetailUri, recipe_id)
	result := BlizzardApi.Recipe{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(helper.api, region, basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}, profession_recipe_uri, getNamespace(static_ns, region), &result)
	if fetchErr != nil {
		return BlizzardApi.Recipe{}, fetchErr
	}
	cache_provider.CacheSet(helper.cache, PROFESSION_RECIPE_DETAIL_CACHE, key, &result, cache_provider.GetStaticTimeWithShift())
	return result, nil
}

// Return an auction house for a given realm and region from the Blizzard API
func (helper *BlizzardApiHelper) GetAuctionHouse(server_id globalTypes.ConnectedRealmID, server_region globalTypes.RegionCode) (BlizzardApi.Auctions, error) {
	key := fmt.Sprint(server_id)

	if found, err := cache_provider.CacheCheck(helper.cache, AUCTION_DATA_CACHE, key); err == nil && found {
		item := BlizzardApi.Auctions{}
		fndErr := cache_provider.CacheGet(helper.cache, AUCTION_DATA_CACHE, key, &item)
		return item, fndErr
	}

	auction_house_fetch_uri := fmt.Sprintf(getAuctionHouseUri, server_id)
	result := BlizzardApi.Auctions{}
	_, fetchErr := blizzard_api_call.GetBlizzardAPIResponse(helper.api, server_region, basicDataPackage{}, auction_house_fetch_uri, getNamespace(dynamic_ns, server_region), &result)
	if fetchErr != nil {
		return BlizzardApi.Auctions{}, fetchErr
	}
	cache_provider.CacheSet(helper.cache, AUCTION_DATA_CACHE, key, &result, time.Duration(time.Hour*1))
	return result, nil
}

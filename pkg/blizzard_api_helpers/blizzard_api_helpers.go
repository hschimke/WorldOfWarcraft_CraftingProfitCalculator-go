package blizzard_api_helpers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"golang.org/x/sync/errgroup"
)

const (
	exclude_before_shadowlands           bool   = false
	static_ns                            string = "static"
	dynamic_ns                           string = "dynamic"
	searchPageSize                       string = "1000"
	ITEM_SEARCH_CACHE                    string = "item_search_cache"
	CONNECTED_REALM_ID_CACHE             string = "connected_realm_data"
	ITEM_DATA_CACHE                      string = "fetched_item_data"
	PROFESSION_SKILL_TIER_DETAILS_CACHE  string = "fetched_profession_skill_tier_detail_data"
	PROFESSION_RECIPE_DETAIL_CACHE       string = "fetched_profession_recipe_detail_data"
	CRAFTABLE_BY_PROFESSION_SET_CACHE    string = "craftable_by_professions_cache"
	CRAFTABLE_BY_SINGLE_PROFESSION_CACHE string = "craftable_by_profession"
	AUCTION_DATA_CACHE                   string = "fetched_auctions_data"
	PROFESSION_DETAIL_CACHE              string = "profession_detail_data"
	PROFESSION_LIST_CACHE                string = "regional_profession_list"
	COMPOSITE_REALM_NAME_CACHE           string = "connected_realm_detail"
	CYCLIC_LINK_CACHE                    string = "cyclic_links"
	ALL_REALM_NAMES_CACHE                string = "all_realm_names"
)

type basicDataPackage map[string]string
type searchDataPackage map[string]string
type searchPageDataPackage map[string]string

type skilltier struct {
	Name string `json:"name,omitempty"`
	Id   uint   `json:"id,omitempty"`
}

type SkillTierCyclicLinksBuild [][]struct {
	Id       []uint
	Quantity uint
}

// Check if a page of search results contains an item named itemName. foundItemId can be ignored if found is false
func checkPageSearchResults(page BlizzardApi.ItemSearch, itemName globalTypes.ItemName) (foundItemId globalTypes.ItemID, found bool) {
	foundItemId = 0
	found = false
	for _, result := range page.Results {
		if strings.EqualFold(itemName, result.Data.Name[blizzard_api_call.ENGLISH_US]) {
			foundItemId = result.Data.Id
			found = true
			break
		}
	}
	return foundItemId, found
}

// Find the item ID for an item with name itemName
func (helper *BlizzardApiHelper) GetItemId(ctx context.Context, region globalTypes.RegionCode, itemName globalTypes.ItemName) (globalTypes.ItemID, error) {
	if found, err := cache_provider.CacheCheck(helper.cache, ITEM_SEARCH_CACHE, itemName); err == nil && found {
		item := globalTypes.ItemID(0)
		fndErr := cache_provider.CacheGet(helper.cache, ITEM_SEARCH_CACHE, itemName, &item)
		return item, fndErr
	}

	item_id := uint(0)

	page_count := uint(0)
	const search_api_uri = "/data/wow/search/item"

	fetchPage := BlizzardApi.ItemSearch{}
	err := blizzard_api_call.GetBlizzardAPIResponse(ctx, helper.api, region, searchDataPackage{
		"locale":     blizzard_api_call.ENGLISH_US,
		"name.en_US": itemName,
		"orderby":    "id:desc",
		"_pageSize":  searchPageSize,
	}, search_api_uri, getNamespace(static_ns, region), &fetchPage)
	if err != nil {
		return 0, fmt.Errorf("search error for %s: %w", itemName, err)
	}
	page_count = fetchPage.PageCount

	helper.logger.Debug("Found ", page_count, " pages for item search ", itemName)
	if page_count > 0 {
		if page_item_id, itemFound := checkPageSearchResults(fetchPage, itemName); itemFound {
			item_id = page_item_id
		} else {
			for cp := fetchPage.Page + 1; cp <= page_count; cp++ {
				helper.logger.Silly("Checking page ", cp, " for ", itemName)
				getPage := BlizzardApi.ItemSearch{}
				err := blizzard_api_call.GetBlizzardAPIResponse(ctx, helper.api, region, searchPageDataPackage{
					"locale":     blizzard_api_call.ENGLISH_US,
					"name.en_US": itemName,
					"orderby":    "id:desc",
					"_pageSize":  searchPageSize,
					"_page":      fmt.Sprint(cp),
				}, search_api_uri, getNamespace(static_ns, region), &getPage)
				if err != nil {
					return 0, err
				}
				if page_item_id, itemFound := checkPageSearchResults(getPage, itemName); itemFound {
					item_id = page_item_id
					helper.logger.Debug("Found ", item_id, " for ", itemName, " on page ", cp, " of ", page_count)
					break
				}
			}
		}
	} else {
		// We didn't get any results, that's an error
		helper.logger.Error("No items match search ", itemName)
		return 0, fmt.Errorf("no items match search %s", itemName)
	}

	if item_id == 0 {
		return 0, fmt.Errorf("no exact match found for %s", itemName)
	}

	cache_provider.CacheSet(helper.cache, ITEM_SEARCH_CACHE, itemName, item_id, cache_provider.GetStaticTimeWithShift())

	return item_id, nil
}

// Get a list of all connected realms
func (helper *BlizzardApiHelper) getAllConnectedRealms(ctx context.Context, region globalTypes.RegionCode) (BlizzardApi.ConnectedRealmIndex, error) {

	const list_connected_realms_api string = "/data/wow/connected-realm/index"
	list_connected_realms_form := basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}

	var realm_index BlizzardApi.ConnectedRealmIndex
	fetchError := blizzard_api_call.GetBlizzardAPIResponse(ctx, helper.api, region, list_connected_realms_form, list_connected_realms_api, getNamespace(dynamic_ns, region), &realm_index)
	if fetchError != nil {
		return BlizzardApi.ConnectedRealmIndex{}, fetchError
	}

	return realm_index, nil
}

// Get a blizzard connected realm based on its ID
func (helper *BlizzardApiHelper) GetConnectedRealmId(ctx context.Context, server_name globalTypes.RealmName, server_region globalTypes.RegionCode) (globalTypes.ConnectedRealmID, error) {
	connected_realm_key := fmt.Sprintf("%s::%s", server_region, server_name)

	if found, err := cache_provider.CacheCheck(helper.cache, CONNECTED_REALM_ID_CACHE, connected_realm_key); err == nil && found {
		item := globalTypes.ConnectedRealmID(0)
		fndErr := cache_provider.CacheGet(helper.cache, CONNECTED_REALM_ID_CACHE, connected_realm_key, &item)
		return item, fndErr
	}

	get_connected_realm_form := basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}

	realm_id := globalTypes.ConnectedRealmID(0)

	// Get a list of all connected realms
	all_connected_realms, err := helper.getAllConnectedRealms(ctx, server_region)
	if err != nil {
		return globalTypes.ConnectedRealmID(0), err
	}

	// Pull the data for each connection until you find one with the server name in question
	for _, realm_href := range all_connected_realms.Connected_realms {
		hr := realm_href.Href
		var connected_realm_detail BlizzardApi.ConnectedRealm
		fetchErr := blizzard_api_call.GetBlizzardRawUriResponse(ctx, helper.api, get_connected_realm_form, hr, server_region, getNamespace(dynamic_ns, server_region), &connected_realm_detail)
		if fetchErr != nil {
			return globalTypes.ConnectedRealmID(0), fetchErr
		}

		found_realm := false
		for _, rlm := range connected_realm_detail.Realms {
			helper.logger.Debugf("Realm %s", rlm.Name)
			if strings.EqualFold(server_name, rlm.Name) {
				helper.logger.Debugf("Realm %v matches %s", rlm, server_name)
				found_realm = true
				break
			}
		}
		if found_realm {
			realm_id = connected_realm_detail.Id
			break
		}
	}

	if realm_id == 0 {
		return 0, fmt.Errorf("realm %s could not be resolved", server_name)
	}

	cache_provider.CacheSet(helper.cache, CONNECTED_REALM_ID_CACHE, connected_realm_key, realm_id, cache_provider.GetStaticTimeWithShift())
	helper.logger.Infof("Found Connected Realm ID: %d for %s %s", realm_id, server_region, server_name)

	// Return that connected realm ID
	return realm_id, nil
}

// Check whether an item is craftable with a given set of professions.
func (helper *BlizzardApiHelper) CheckIsCrafting(ctx context.Context, item_id globalTypes.ItemID, character_professions []globalTypes.CharacterProfession, region globalTypes.RegionCode, static_source *static_sources.StaticSources) (globalTypes.CraftingStatus, error) {
	// Check if we've already run this check, and if so return the cached version, otherwise keep on
	key := fmt.Sprintf("%s::%d::%v", region, item_id, character_professions)

	if found, err := cache_provider.CacheCheck(helper.cache, CRAFTABLE_BY_PROFESSION_SET_CACHE, key); err == nil && found {
		item := globalTypes.CraftingStatus{}
		fndErr := cache_provider.CacheGet(helper.cache, CRAFTABLE_BY_PROFESSION_SET_CACHE, key, &item)
		return item, fndErr
	}

	profession_list, err := helper.GetBlizProfessionsList(ctx, region)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}

	recipe_options := globalTypes.CraftingStatus{
		Craftable: false,
	}

	// Check if a vendor is mentioned in the item description and if so just short circuit
	item_detail, err := helper.GetItemDetails(ctx, item_id, region)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}
	if item_detail.Description != "" {
		if strings.Contains(item_detail.Description, "vendor") {
			helper.logger.Debug("Skipping vendor recipe")
			cache_provider.CacheSet(helper.cache, CRAFTABLE_BY_PROFESSION_SET_CACHE, key, &recipe_options, cache_provider.GetComputedTimeWithShift())
			return recipe_options, nil
		}
	}

	var profession_result_array []globalTypes.CraftingStatus
	var mutex sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)

	for _, prof := range character_professions {
		prof := prof
		if profession_id, profErr := getProfessionId(profession_list, prof); profErr == nil {
			g.Go(func() error {
				res, err := helper.checkProfessionCrafting(gCtx, profession_id, prof, region, item_id, item_detail, static_source)
				if err != nil {
					return err
				}
				mutex.Lock()
				profession_result_array = append(profession_result_array, res)
				mutex.Unlock()
				return nil
			})
		} else {
			helper.logger.Warnf("Could not find profession ID for %s: %v", prof, profErr)
		}
	}

	if err := g.Wait(); err != nil {
		return globalTypes.CraftingStatus{}, err
	}

	// collate professions
	for _, profession_crafting_check := range profession_result_array {
		recipe_options.Recipes = append(recipe_options.Recipes, profession_crafting_check.Recipes...)
		recipe_options.Recipe_ids = append(recipe_options.Recipe_ids, profession_crafting_check.Recipe_ids...)
		recipe_options.Craftable = recipe_options.Craftable || profession_crafting_check.Craftable
	}

	cache_provider.CacheSet(helper.cache, CRAFTABLE_BY_PROFESSION_SET_CACHE, key, &recipe_options, cache_provider.GetComputedTimeWithShift())
	return recipe_options, nil
}

// Find the ID of a profession given a list of professions and a profession name
func getProfessionId(profession_list BlizzardApi.ProfessionsIndex, profession_name string) (uint, error) {
	var id uint = 0
	for _, prof := range profession_list.Professions {
		if prof.Name == profession_name {
			id = prof.Id
			break
		}
	}
	if id == 0 {
		return 0, fmt.Errorf("could not find profession id for %s", profession_name)
	}
	return id, nil
}

// Check whether and item can be crafted by a given skilltier within a profession
func (helper *BlizzardApiHelper) checkProfessionTierCrafting(ctx context.Context, skill_tier skilltier, region globalTypes.RegionCode, item_id uint, check_profession_id uint, prof string, item_detail BlizzardApi.Item, profession_recipe_options *globalTypes.CraftingStatus, mutex *sync.Mutex, static_source *static_sources.StaticSources) error {
	check_scan_tier := true
	if exclude_before_shadowlands {
		check_scan_tier = strings.Contains(skill_tier.Name, "Shadowlands")
	}

	if !check_scan_tier {
		return nil
	}

	helper.logger.Debugf("Checking: %s for: %d", skill_tier.Name, item_id)
	// Get a list of all recipes each level can do
	skill_tier_detail, err := helper.GetBlizSkillTierDetail(ctx, check_profession_id, skill_tier.Id, region)
	if err != nil {
		return err
	}

	if skill_tier_detail.Categories == nil {
		helper.logger.Debugf("Skill tier %s has no categories.", skill_tier.Name)
		return nil
	}

	g, gCtx := errgroup.WithContext(ctx)

	for _, cat := range skill_tier_detail.Categories {
		cat := cat
		for _, rec := range cat.Recipes {
			rec := rec
			g.Go(func() error {
				recipe, err := helper.GetBlizRecipeDetail(gCtx, rec.Id, region)
				if err != nil {
					return err
				}

				if strings.Contains(recipe.Name, "Prospect") || strings.Contains(recipe.Name, "Mill") {
					return nil
				}

				crafty := false
				ids := getRecipeCraftedItemID(gCtx, recipe, region, helper, static_source)

				for _, id := range ids {
					if id == item_id {
						crafty = true
					}
				}

				// Enchantments fallback
				if !crafty && (strings.Contains(skill_tier.Name, "Enchanting") && (strings.Contains(cat.Name, "Enchantments") || strings.Contains(cat.Name, "Echantments"))) {
					slot := getSlotName(cat)
					synthetic_item_name := fmt.Sprintf("Enchant %s - %s", slot, rec.Name)
					synthetic_item_id, err := helper.GetItemId(gCtx, region, synthetic_item_name)
					if err == nil && synthetic_item_id != 0 && synthetic_item_id == item_id {
						crafty = true
					}
				}

				if crafty {
					mutex.Lock()
					helper.logger.Infof("Found recipe (%d): %s for (%d) %s", recipe.Id, recipe.Name, item_detail.Id, item_detail.Name)
					profession_recipe_options.Recipes = append(profession_recipe_options.Recipes, struct {
						Recipe_id           uint
						Crafting_profession string
					}{
						recipe.Id,
						prof,
					})
					profession_recipe_options.Recipe_ids = append(profession_recipe_options.Recipe_ids, recipe.Id)
					profession_recipe_options.Craftable = true
					mutex.Unlock()
				}
				return nil
			})
		}
	}

	return g.Wait()
}

// Check whether an item can be crafted by a given profession
func (helper *BlizzardApiHelper) checkProfessionCrafting(ctx context.Context, profession_id uint, prof globalTypes.CharacterProfession, region globalTypes.RegionCode, item_id globalTypes.ItemID, item_detail BlizzardApi.Item, static_source *static_sources.StaticSources) (globalTypes.CraftingStatus, error) {
	cache_key := fmt.Sprintf("%s:%s:%d", region, prof, item_id)
	if found, err := cache_provider.CacheCheck(helper.cache, CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key); err == nil && found {
		item := globalTypes.CraftingStatus{}
		fndErr := cache_provider.CacheGet(helper.cache, CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key, &item)
		return item, fndErr
	}
	profession_recipe_options := globalTypes.CraftingStatus{}
	profession_recipe_options.Craftable = false

	profession_detail, err := helper.GetBlizProfessionDetail(ctx, profession_id, region)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}

	helper.logger.Debug("Scanning profession: ", profession_detail.Name)

	g, gCtx := errgroup.WithContext(ctx)
	var lock sync.Mutex

	for _, tier := range profession_detail.Skill_tiers {
		tier := tier
		st := skilltier{Name: tier.Name, Id: tier.Id}
		g.Go(func() error {
			return helper.checkProfessionTierCrafting(gCtx, st, region, uint(item_id), profession_id, string(prof), item_detail, &profession_recipe_options, &lock, static_source)
		})
	}

	if err := g.Wait(); err != nil {
		return globalTypes.CraftingStatus{}, err
	}

	cache_provider.CacheSet(helper.cache, CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key, profession_recipe_options, cache_provider.GetComputedTimeWithShift())

	return profession_recipe_options, nil

}

// Get the ID of an item crafted by a given recipe. If multiple items are crafted return them all
func getRecipeCraftedItemID(ctx context.Context, recipe BlizzardApi.Recipe, region globalTypes.RegionCode, helper *BlizzardApiHelper, static_source *static_sources.StaticSources) []globalTypes.ItemID {
	item_ids := make(map[globalTypes.ItemID]bool)

	found := false
	if recipe.Horde_crafted_item != nil {
		item_ids[recipe.Horde_crafted_item.Id] = true
		found = true
	}
	if recipe.Alliance_crafted_item != nil {
		item_ids[recipe.Alliance_crafted_item.Id] = true
		found = true
	}
	if recipe.Crafted_item != nil {
		item_ids[recipe.Crafted_item.Id] = true
		found = true
	}

	if !found {
		searched_id, search_id_err := helper.GetItemId(ctx, region, recipe.Name)
		if search_id_err == nil {
			item_ids[searched_id] = true
			found = true
		}
		if !found {
			firesong_link_table, fetch_err := static_source.GetFireSongsCraftingLinkTable()
			if fetch_err == nil {
				for _, element := range *firesong_link_table {
					if element.RecipeId == recipe.Id {
						item_ids[element.Id] = true
						found = true
					}
				}
			}
		}
	}

	if !found {
		return make([]globalTypes.ItemID, 0)
	}

	return_ids := make([]globalTypes.ItemID, 0, len(item_ids))
	for key := range item_ids {
		return_ids = append(return_ids, key)
	}

	return return_ids
}

// Find what the equipment slot name of an enchantment should be
func getSlotName(category BlizzardApi.Category) (raw_slot_name string) {
	var name string = category.Name

	raw_slot_name = name

	if loc_sp := strings.Index(name, "Enchantments"); loc_sp > 0 {
		raw_slot_name = name[:loc_sp-1]
	} else if loc_incsp := strings.Index(name, "Echantments"); loc_incsp > 0 {
		raw_slot_name = name[:loc_incsp-1]
	}

	switch raw_slot_name {
	case "Boot":
		raw_slot_name = "Boots"
	case "Glove":
		raw_slot_name = "Gloves"
	}

	return raw_slot_name
}

// Construct a list of cyclic links between recipes
func (helper *BlizzardApiHelper) BuildCyclicRecipeList(ctx context.Context, region globalTypes.RegionCode, static_source *static_sources.StaticSources) (globalTypes.SkillTierCyclicLinks, error) {
	profession_list, err := helper.GetBlizProfessionsList(ctx, region)
	if err != nil {
		return globalTypes.SkillTierCyclicLinks{}, err
	}

	var links SkillTierCyclicLinksBuild
	var linksMutex sync.Mutex
	var counter uint64

	g, gCtx := errgroup.WithContext(ctx)

	for _, pro := range profession_list.Professions {
		pro := pro
		g.Go(func() error {
			profession, err := helper.GetBlizProfessionDetail(gCtx, pro.Id, region)
			if err != nil {
				return err
			}
			
			sg, sgCtx := errgroup.WithContext(gCtx)
			for _, st := range profession.Skill_tiers {
				st := st
				sg.Go(func() error {
					data, new_count := helper.buildCyclicLinkforSkillTier(sgCtx, skilltier{Name: st.Name, Id: st.Id}, profession, region, static_source)
					linksMutex.Lock()
					links = append(links, data...)
					atomic.AddUint64(&counter, new_count)
					linksMutex.Unlock()
					return nil
				})
			}
			return sg.Wait()
		})
	}

	if err := g.Wait(); err != nil {
		return globalTypes.SkillTierCyclicLinks{}, err
	}

	link_lookup := make(globalTypes.SkillTierCyclicLinks)

	for _, link := range links {
		item_1 := link[0]
		item_2 := link[1]

		// Item 1 links
		for _, id_1 := range item_1.Id {
			if _, present := link_lookup[id_1]; !present {
				link_lookup[id_1] = make([]struct {
					Id    uint
					Takes float64
					Makes float64
				}, 0)
			}
			for _, id_2 := range item_2.Id {
				if id_1 != id_2 {
					link_lookup[id_1] = append(link_lookup[id_1], struct {
						Id    uint
						Takes float64
						Makes float64
					}{
						id_2,
						float64(item_2.Quantity),
						float64(item_1.Quantity),
					})
				}
			}
		}

		// Item 2 links
		for _, id_2 := range item_2.Id {
			if _, present := link_lookup[id_2]; !present {
				link_lookup[id_2] = make([]struct {
					Id    uint
					Takes float64
					Makes float64
				}, 0)
			}
			for _, id_1 := range item_1.Id {
				if id_2 != id_1 {
					link_lookup[id_2] = append(link_lookup[id_2], struct {
						Id    uint
						Takes float64
						Makes float64
					}{
						id_1,
						float64(item_1.Quantity),
						float64(item_2.Quantity),
					})
				}
			}
		}
	}

	return link_lookup, nil
}

// Check of cyclic links within a single skill tier
func (helper *BlizzardApiHelper) buildCyclicLinkforSkillTier(ctx context.Context, skill_tier skilltier, profession BlizzardApi.Profession, region globalTypes.RegionCode, static_source *static_sources.StaticSources) (SkillTierCyclicLinksBuild, uint64) {
	cache_key := fmt.Sprintf("%s::%s::%d", region, skill_tier.Name, profession.Id)

	if found, err := cache_provider.CacheCheck(helper.cache, CYCLIC_LINK_CACHE, cache_key); err == nil && found {
		var item SkillTierCyclicLinksBuild
		cache_provider.CacheGet(helper.cache, CYCLIC_LINK_CACHE, cache_key, &item)
		return item, 0
	}
	var counter uint64
	var found_links SkillTierCyclicLinksBuild
	skill_tier_detail, err := helper.GetBlizSkillTierDetail(ctx, profession.Id, skill_tier.Id, region)
	if err != nil {
		return SkillTierCyclicLinksBuild{}, 0
	}
	
	// This is a heavy operation, but we'll keep it simple for now and rely on parent context for cancellation
	var checked_set util.Set[uint]
	for _, sk_category := range skill_tier_detail.Categories {
		for _, sk_recipe := range sk_category.Recipes {
			recipe, err := helper.GetBlizRecipeDetail(ctx, sk_recipe.Id, region)
			if err != nil {
				return SkillTierCyclicLinksBuild{}, 0
			}
			if !checked_set.Has(recipe.Id) {
				checked_set.Add(recipe.Id)
				counter++
				if len(recipe.Reagents) == 1 {
					for _, sk_recheck_category := range skill_tier_detail.Categories {
						for _, sk_recheck_recipe := range sk_recheck_category.Recipes {
							recheck_recipe, err := helper.GetBlizRecipeDetail(ctx, sk_recheck_recipe.Id, region)
							if err != nil {
								return SkillTierCyclicLinksBuild{}, 0
							}
							if len(recheck_recipe.Reagents) == 1 && !checked_set.Has(recheck_recipe.Id) {
								r_ids := getRecipeCraftedItemID(ctx, recipe, region, helper, static_source)
								rc_ids := getRecipeCraftedItemID(ctx, recheck_recipe, region, helper, static_source)

								if slices.Contains(r_ids, recheck_recipe.Reagents[0].Reagent.Id) {
									if slices.Contains(rc_ids, recipe.Reagents[0].Reagent.Id) {
										p1 := getRecipeCraftedItemID(ctx, recipe, region, helper, static_source)
										p2 := getRecipeCraftedItemID(ctx, recheck_recipe, region, helper, static_source)
										found_links = append(found_links, []struct {
											Id       []uint
											Quantity uint
										}{
											{Id: p1, Quantity: recheck_recipe.Reagents[0].Quantity},
											{Id: p2, Quantity: recipe.Reagents[0].Quantity},
										})
										checked_set.Add(recheck_recipe.Id)
									}
								}
							} else {
								checked_set.Add(recheck_recipe.Id)
							}
						}
					}
				}
			}
		}
	}
	cache_provider.CacheSet(helper.cache, CYCLIC_LINK_CACHE, cache_key, found_links, cache_provider.GetStaticTimeWithShift())
	return found_links, counter
}

// Construct the blizzard namespace for a given region and type (static or dynamic)
func getNamespace(ns_type string, region globalTypes.RegionCode) string {
	return fmt.Sprintf("%s-%s", ns_type, strings.ToLower(string(region)))
}

// Return a list of all realm names
func (helper *BlizzardApiHelper) GetAllRealmNames(ctx context.Context, region globalTypes.RegionCode) []string {
	all_realm_key := string(region)

	var realmNames []string

	if found, err := cache_provider.CacheCheck(helper.cache, ALL_REALM_NAMES_CACHE, all_realm_key); err == nil && found {
		cache_provider.CacheGet(helper.cache, ALL_REALM_NAMES_CACHE, all_realm_key, &realmNames)
		return realmNames
	}

	get_connected_realm_form := basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}

	// Get a list of all connected realms
	all_connected_realms, err := helper.getAllConnectedRealms(ctx, region)
	if err != nil {
		return realmNames
	}

	// Pull the data for each connection until you find one with the server name in question
	for _, realm_href := range all_connected_realms.Connected_realms {
		hr := realm_href.Href
		var connected_realm_detail BlizzardApi.ConnectedRealm
		fetchErr := blizzard_api_call.GetBlizzardRawUriResponse(ctx, helper.api, get_connected_realm_form, hr, region, getNamespace(dynamic_ns, region), &connected_realm_detail)
		if fetchErr != nil {
			return realmNames
		}

		for _, rlm := range connected_realm_detail.Realms {
			realmNames = append(realmNames, rlm.Name)
		}
	}

	cache_provider.CacheSet(helper.cache, ALL_REALM_NAMES_CACHE, all_realm_key, realmNames, cache_provider.GetDynamicTimeWithShift())

	return realmNames
}

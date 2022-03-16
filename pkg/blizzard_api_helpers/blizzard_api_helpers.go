package blizzard_api_helpers

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"golang.org/x/exp/slices"
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
func GetItemId(region globalTypes.RegionCode, itemName globalTypes.ItemName) (globalTypes.ItemID, error) {
	if found, err := cache_provider.CacheCheck(ITEM_SEARCH_CACHE, itemName); err == nil && found {
		item := globalTypes.ItemID(0)
		fndErr := cache_provider.CacheGet(ITEM_SEARCH_CACHE, itemName, &item)
		return item, fndErr
	}

	item_id := uint(0)

	//current_page := uint(0)
	page_count := uint(0)
	const search_api_uri = "/data/wow/search/item"

	fetchPage := BlizzardApi.ItemSearch{}
	_, err := blizzard_api_call.GetBlizzardAPIResponse(region, searchDataPackage{
		"locale":     blizzard_api_call.ENGLISH_US,
		"name.en_US": itemName,
		"orderby":    "id:desc",
		"_pageSize":  searchPageSize,
	}, search_api_uri, getNamespace(static_ns, region), &fetchPage)
	//current_page = fetchPage.Page
	if err != nil && fetchPage.PageCount <= 0 {
		return 0, fmt.Errorf("no results for %s", itemName)
	}
	page_count = fetchPage.PageCount
	//return fetchPage, fetchPage.Page, fetchPage.PageCount, err

	cpclog.Debug("Found ", page_count, " pages for item search ", itemName)
	if page_count > 0 {
		if page_item_id, itemFound := checkPageSearchResults(fetchPage, itemName); itemFound {
			item_id = page_item_id
		} else {
			for cp := fetchPage.Page; cp <= page_count; cp++ {
				cpclog.Silly("Checking page ", cp, " for ", itemName)
				getPage := BlizzardApi.ItemSearch{}
				_, err := blizzard_api_call.GetBlizzardAPIResponse(region, searchPageDataPackage{
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
					cpclog.Debug("Found ", item_id, " for ", itemName, " on page ", cp, " of ", page_count)
					break
				}
			}
		}
	} else {
		// We didn't get any results, that's an error
		cpclog.Error("No items match search ", itemName)
		return 0, fmt.Errorf("no items match search %s", itemName)
	}

	cache_provider.CacheSet(ITEM_SEARCH_CACHE, itemName, item_id, cache_provider.GetStaticTimeWithShift())

	return item_id, nil
}

// Get a list of all connected realms
func getAllConnectedRealms(region globalTypes.RegionCode) (BlizzardApi.ConnectedRealmIndex, error) {

	const list_connected_realms_api string = "/data/wow/connected-realm/index"
	list_connected_realms_form := basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}

	var realm_index BlizzardApi.ConnectedRealmIndex
	_, fetchError := blizzard_api_call.GetBlizzardAPIResponse(region, list_connected_realms_form, list_connected_realms_api, getNamespace(dynamic_ns, region), &realm_index)
	if fetchError != nil {
		return BlizzardApi.ConnectedRealmIndex{}, fetchError
	}

	return realm_index, nil
}

// Get a blizzard connected realm based on its ID
func GetConnectedRealmId(server_name globalTypes.RealmName, server_region globalTypes.RegionCode) (globalTypes.ConnectedRealmID, error) {
	connected_realm_key := fmt.Sprintf("%s::%s", server_region, server_name)

	if found, err := cache_provider.CacheCheck(CONNECTED_REALM_ID_CACHE, connected_realm_key); err == nil && found {
		item := globalTypes.ConnectedRealmID(0)
		fndErr := cache_provider.CacheGet(CONNECTED_REALM_ID_CACHE, connected_realm_key, &item)
		return item, fndErr
	}

	get_connected_realm_form := basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}

	realm_id := globalTypes.ConnectedRealmID(0)

	// Get a list of all connected realms
	all_connected_realms, err := getAllConnectedRealms(server_region)
	if err != nil {
		return globalTypes.ConnectedRealmID(0), err
	}

	// Pull the data for each connection until you find one with the server name in question
	for _, realm_href := range all_connected_realms.Connected_realms {
		hr := realm_href.Href
		var connected_realm_detail BlizzardApi.ConnectedRealm
		_, crErr := blizzard_api_call.GetBlizzardRawUriResponse(get_connected_realm_form, hr, server_region, getNamespace(dynamic_ns, server_region), &connected_realm_detail)
		if crErr != nil {
			return globalTypes.ConnectedRealmID(0), crErr
		}

		found_realm := false
		for _, rlm := range connected_realm_detail.Realms {
			cpclog.Debugf("Realm %s", rlm.Name)
			if strings.EqualFold(server_name, rlm.Name) {
				cpclog.Debugf("Realm %v matches %s", rlm, server_name)
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

	cache_provider.CacheSet(CONNECTED_REALM_ID_CACHE, connected_realm_key, realm_id, cache_provider.GetStaticTimeWithShift())
	cpclog.Infof("Found Connected Realm ID: %d for %s %s", realm_id, server_region, server_name)

	// Return that connected realm ID
	return realm_id, nil
}

// Check whether an item is craftable with a given set of professions.
func CheckIsCrafting(item_id globalTypes.ItemID, character_professions []globalTypes.CharacterProfession, region globalTypes.RegionCode) (globalTypes.CraftingStatus, error) {
	// Check if we've already run this check, and if so return the cached version, otherwise keep on
	key := fmt.Sprintf("%s::%d::%v", region, item_id, character_professions)

	if found, err := cache_provider.CacheCheck(CRAFTABLE_BY_PROFESSION_SET_CACHE, key); err == nil && found {
		item := globalTypes.CraftingStatus{}
		fndErr := cache_provider.CacheGet(CRAFTABLE_BY_PROFESSION_SET_CACHE, key, &item)
		return item, fndErr
	}

	profession_list, err := GetBlizProfessionsList(region)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}

	recipe_options := globalTypes.CraftingStatus{
		Craftable: false,
	}

	// Check if a vendor is mentioned in the item description and if so just short circuit
	item_detail, err := GetItemDetails(item_id, region)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}
	if item_detail.Description != "" {
		if strings.Contains(item_detail.Description, "vendor") {
			cpclog.Debug("Skipping vendor recipe")
			cache_provider.CacheSet(CRAFTABLE_BY_PROFESSION_SET_CACHE, key, &recipe_options, cache_provider.GetComputedTimeWithShift())
			return recipe_options, nil
		}
	}

	var profession_result_array []globalTypes.CraftingStatus

	type iData struct {
		profession_id uint
		prof          globalTypes.CharacterProfession
		region        globalTypes.RegionCode
		item_id       globalTypes.ItemID
		item_detail   BlizzardApi.Item
	}

	type iRet struct {
		ret globalTypes.CraftingStatus
		err error
	}

	inputData := make(chan iData, 3)
	outputData := make(chan iRet, len(character_professions))

	workerFunc := func(input chan iData, output chan iRet) {
		for z := range input {
			z, err := checkProfessionCrafting(z.profession_id, z.prof, z.region, z.item_id, z.item_detail)
			output <- iRet{z, err}
		}
	}

	var errArr []error

	for i := 0; i < len(character_professions); i++ {
		go workerFunc(inputData, outputData)
	}

	for _, prof := range character_professions {
		if profession_id, profErr := getProfessionId(profession_list, prof); profErr == nil {
			inputData <- iData{profession_id, prof, region, item_id, item_detail}
		} else {
			errArr = append(errArr, profErr)
		}
	}
	close(inputData)

	for job := 1; job <= len(character_professions); job++ {
		jBr := <-outputData
		if jBr.err != nil {
			errArr = append(errArr, jBr.err)
		}
		profession_result_array = append(profession_result_array, jBr.ret)
	}

	if len(errArr) != 0 {
		var str strings.Builder

		for i, e := range errArr {
			str.WriteString(fmt.Sprintf("error %d: %v\n", i, e))
		}
		return globalTypes.CraftingStatus{}, errors.New(str.String())
	}

	// collate professions
	for _, profession_crafting_check := range profession_result_array {
		recipe_options.Recipes = append(recipe_options.Recipes, profession_crafting_check.Recipes...)
		recipe_options.Recipe_ids = append(recipe_options.Recipe_ids, profession_crafting_check.Recipe_ids...)
		recipe_options.Craftable = recipe_options.Craftable || profession_crafting_check.Craftable
	}

	cache_provider.CacheSet(CRAFTABLE_BY_PROFESSION_SET_CACHE, key, &recipe_options, cache_provider.GetComputedTimeWithShift())
	//{craftable: found_craftable, recipe_id: found_recipe_id, crafting_profession: found_profession};
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
func checkProfessionTierCrafting(skill_tier skilltier, region globalTypes.RegionCode, item_id uint, check_profession_id uint, prof string, item_detail BlizzardApi.Item, profession_recipe_options *globalTypes.CraftingStatus, mutex *sync.Mutex) {
	check_scan_tier := true
	if exclude_before_shadowlands {
		check_scan_tier = strings.Contains(skill_tier.Name, "Shadowlands")
	}

	if check_scan_tier {
		cpclog.Debugf("Checking: %s for: %d", skill_tier.Name, item_id)
		// Get a list of all recipes each level can do
		skill_tier_detail, err := GetBlizSkillTierDetail(check_profession_id, skill_tier.Id, region)
		if err != nil {
			return
		}

		checked_categories := 0
		recipes_checked := 0

		if skill_tier_detail.Categories != nil {
			categories := skill_tier_detail.Categories

			checked_categories += len(categories)
			for _, cat := range categories {
				for _, rec := range cat.Recipes {
					recipe, err := GetBlizRecipeDetail(rec.Id, region)
					if err != nil {
						return
					}
					recipes_checked++
					cpclog.Sillyf("Check recipe %s", recipe.Name)
					if !(strings.Contains(recipe.Name, "Prospect") || strings.Contains(recipe.Name, "Mill")) {
						crafty := false
						ids := getRecipeCraftedItemID(recipe)

						for _, id := range ids {
							if id == item_id {
								crafty = true
							}
						}

						// Enchantments and Echantments must be checked because of a onetime data error in the API response
						if !crafty && (strings.Contains(skill_tier.Name, "Enchanting") && (strings.Contains(cat.Name, "Enchantments") || strings.Contains(cat.Name, "Echantments"))) {
							cpclog.Sillyf("Checking if uncraftable item %d is craftable with a synthetic item-recipe connection.", item_detail.Id)
							slot := getSlotName(cat)
							synthetic_item_name := fmt.Sprintf("Enchant %s - %s", slot, rec.Name)
							cpclog.Sillyf("Generated synthetic item name ", synthetic_item_name)
							synthetic_item_id, err := GetItemId(region, synthetic_item_name)
							if err != nil {
								return
							}
							cpclog.Sillyf("Synthetic item %s has id %s", synthetic_item_name, synthetic_item_id)
							if synthetic_item_id != 0 && synthetic_item_id == item_id {
								crafty = true
								cpclog.Sillyf("Synthetic item %s match for %s.", synthetic_item_name, item_detail.Name)
							}
						} else {
							cpclog.Sillyf("Skipping synthetic for %t (%t) %s (%t) %s (%t) %s", crafty, !crafty, skill_tier.Name, strings.Contains(skill_tier.Name, "Enchanting"), cat.Name, strings.Contains(cat.Name, "Enchantments"), rec.Name)
						}

						// item is craftable
						if crafty {
							mutex.Lock()
							cpclog.Infof("Found recipe (%d): %s for (%d) %s", recipe.Id, recipe.Name, item_detail.Id, item_detail.Name)

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
					} else {
						cpclog.Sillyf("Skipping Recipe: (%d) \"%s\"", recipe.Id, recipe.Name)
					}
				}
			}
		} else {
			cpclog.Debugf("Skill tier %s has no categories.", skill_tier.Name)
		}
		cpclog.Debug("Checked ", recipes_checked, " recipes in ", checked_categories, " categories for ", item_id, " in ", skill_tier.Name)
	}
}

// Check whether an item can be crafted by a given profession
func checkProfessionCrafting(profession_id uint, prof globalTypes.CharacterProfession, region globalTypes.RegionCode, item_id globalTypes.ItemID, item_detail BlizzardApi.Item) (globalTypes.CraftingStatus, error) {
	cache_key := fmt.Sprintf("%s:%s:%d", region, prof, item_id)
	if found, err := cache_provider.CacheCheck(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key); err == nil && found {
		item := globalTypes.CraftingStatus{}
		fndErr := cache_provider.CacheGet(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key, &item)
		return item, fndErr
	}
	profession_recipe_options := globalTypes.CraftingStatus{}
	profession_recipe_options.Craftable = false

	/*check_profession_id, err := getProfessionId(profession_list, prof)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}*/

	// Get a list of the crafting levels for the professions
	profession_detail, err := GetBlizProfessionDetail(profession_id, region)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}
	crafting_levels := profession_detail.Skill_tiers

	cpclog.Debug("Scanning profession: ", profession_detail.Name)

	var wg sync.WaitGroup
	var lock sync.Mutex

	for _, tier := range crafting_levels {
		wg.Add(1)
		st := tier
		go func() {
			defer wg.Done()
			checkProfessionTierCrafting(st, region, item_id, profession_id, prof, item_detail, &profession_recipe_options, &lock)
		}()
	}

	wg.Wait()

	cache_provider.CacheSet(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key, profession_recipe_options, cache_provider.GetComputedTimeWithShift())

	return profession_recipe_options, nil

}

// Get the ID of an item crafted by a given recipe. If multiple items are crafted return them all
func getRecipeCraftedItemID(recipe BlizzardApi.Recipe) []globalTypes.ItemID {
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
func BuildCyclicRecipeList(region globalTypes.RegionCode) (globalTypes.SkillTierCyclicLinks, error) {
	profession_list, err := GetBlizProfessionsList(region)
	if err != nil {
		return globalTypes.SkillTierCyclicLinks{}, err
	}

	var links SkillTierCyclicLinksBuild

	var professionScanList []BlizzardApi.Profession
	for _, pro := range profession_list.Professions {
		profDetail, err := GetBlizProfessionDetail(pro.Id, region)
		if err != nil {
			return globalTypes.SkillTierCyclicLinks{}, err
		}
		professionScanList = append(professionScanList, profDetail)
	}

	counter := uint64(0)
	profession_counter := int64(0)

	type oData struct {
		ret SkillTierCyclicLinksBuild
		err error
	}

	inputData := make(chan BlizzardApi.Profession, 4)
	outputData := make(chan oData, len(professionScanList))

	workerFunc := func(input chan BlizzardApi.Profession, output chan oData) {
		var (
			appendMutex        sync.Mutex
			buildCLSkillTierWG sync.WaitGroup
		)
		for prof := range input {
			//var collectLinks SkillTierCyclicLinksBuild
			return_data := oData{}
			last_count := atomic.LoadUint64(&counter)
			cpclog.Debug("Scanning profession: ", prof.Name, " for cyclic relationships.")
			profession, err := GetBlizProfessionDetail(prof.Id, region)
			if err != nil {
				return_data.err = err
			} else if profession.Skill_tiers != nil {
				for _, st := range profession.Skill_tiers {
					buildCLSkillTierWG.Add(1)
					st := st
					profession := profession
					region := region
					go func() {
						defer buildCLSkillTierWG.Done()
						data, new_count := buildCyclicLinkforSkillTier(st, profession, region)
						appendMutex.Lock()
						return_data.ret = append(return_data.ret, data...)
						atomic.AddUint64(&counter, new_count)
						appendMutex.Unlock()
					}()
				}
				buildCLSkillTierWG.Wait()
			}
			cpclog.Debug("Scanned ", atomic.LoadUint64(&counter)-last_count, " new recipes in ", prof.Name, ".")
			atomic.AddInt64(&profession_counter, 1)
			output <- return_data
		}
	}

	for i := 0; i < len(professionScanList); i++ {
		go workerFunc(inputData, outputData)
	}

	for _, prof := range professionScanList {
		inputData <- prof
	}
	close(inputData)

	var errors []error
	for job := 1; job <= len(professionScanList); job++ {
		jbR := <-outputData
		//for job := range outputData {
		if jbR.err != nil {
			errors = append(errors, jbR.err)
		}

		links = append(links, jbR.ret...)
	}

	if len(errors) > 0 {
		return globalTypes.SkillTierCyclicLinks{}, errors[0]
	}

	cpclog.Debug("Scanned ", counter, " recipes in ", atomic.LoadInt64(&profession_counter), " professions")

	link_lookup := make(globalTypes.SkillTierCyclicLinks)

	for _, link := range links {
		item_1 := link[0]
		item_2 := link[1]

		// Item 1 links
		for _, id_1 := range item_1.Id {
			if _, present := link_lookup[id_1]; present {
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
			if _, present := link_lookup[id_2]; present {
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
func buildCyclicLinkforSkillTier(skill_tier skilltier, profession BlizzardApi.Profession, region globalTypes.RegionCode) (SkillTierCyclicLinksBuild, uint64) {
	cache_key := fmt.Sprintf("%s::%s::%d", region, skill_tier.Name, profession.Id)

	if found, err := cache_provider.CacheCheck(CYCLIC_LINK_CACHE, cache_key); err == nil && found {
		var item SkillTierCyclicLinksBuild
		cache_provider.CacheGet(CYCLIC_LINK_CACHE, cache_key, &item)
		return item, 0
	}
	var counter uint64
	cpclog.Debug("Scanning st: ", skill_tier.Name)
	var checked_set util.Set[uint]
	var found_links SkillTierCyclicLinksBuild
	skill_tier_detail, err := GetBlizSkillTierDetail(profession.Id, skill_tier.Id, region)
	if err != nil {
		return SkillTierCyclicLinksBuild{}, 0
	}
	for _, sk_category := range skill_tier_detail.Categories {
		for _, sk_recipe := range sk_category.Recipes {
			recipe, err := GetBlizRecipeDetail(sk_recipe.Id, region)
			if err != nil {
				return SkillTierCyclicLinksBuild{}, 0
			}
			if !checked_set.Has(recipe.Id) {
				checked_set.Add(recipe.Id)
				counter++
				if len(recipe.Reagents) == 1 {
					// Go through them all again
					for _, sk_recheck_category := range skill_tier_detail.Categories {
						for _, sk_recheck_recipe := range sk_recheck_category.Recipes {
							recheck_recipe, err := GetBlizRecipeDetail(sk_recheck_recipe.Id, region)
							if err != nil {
								return SkillTierCyclicLinksBuild{}, 0
							}
							if len(recheck_recipe.Reagents) == 1 && !checked_set.Has(recheck_recipe.Id) {
								r_ids := getRecipeCraftedItemID(recipe)

								rc_ids := getRecipeCraftedItemID(recheck_recipe)

								if slices.Contains(r_ids, recheck_recipe.Reagents[0].Reagent.Id) {
									if slices.Contains(rc_ids, recipe.Reagents[0].Reagent.Id) {
										cpclog.Debugf("Found cyclic link for %s (%d) and %s (%d)", recipe.Name, recipe.Id, recheck_recipe.Name, recheck_recipe.Id)
										p1 := getRecipeCraftedItemID(recipe)
										p2 := getRecipeCraftedItemID(recheck_recipe)
										found_links = append(found_links, []struct {
											Id       []uint
											Quantity uint
										}{
											{
												Id:       p1,
												Quantity: recheck_recipe.Reagents[0].Quantity,
											},
											{
												Id:       p2,
												Quantity: recipe.Reagents[0].Quantity,
											},
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
	cache_provider.CacheSet(CYCLIC_LINK_CACHE, cache_key, found_links, cache_provider.GetStaticTimeWithShift())
	return found_links, counter
}

// Construct the blizzard namespace for a given region and type (static or dynamic)
func getNamespace(ns_type string, region globalTypes.RegionCode) string {
	return fmt.Sprintf("%s-%s", ns_type, strings.ToLower(region))
}

// Return a list of all realm names
func GetAllRealmNames(region globalTypes.RegionCode) []string {
	all_realm_key := string(region)

	var realmNames []string

	if found, err := cache_provider.CacheCheck(ALL_REALM_NAMES_CACHE, all_realm_key); err == nil && found {
		cache_provider.CacheGet(ALL_REALM_NAMES_CACHE, all_realm_key, &realmNames)
		return realmNames
	}

	get_connected_realm_form := basicDataPackage{
		"locale": blizzard_api_call.ENGLISH_US,
	}

	// Get a list of all connected realms
	all_connected_realms, err := getAllConnectedRealms(region)
	if err != nil {
		return realmNames
	}

	// Pull the data for each connection until you find one with the server name in question
	for _, realm_href := range all_connected_realms.Connected_realms {
		hr := realm_href.Href
		var connected_realm_detail BlizzardApi.ConnectedRealm
		_, crErr := blizzard_api_call.GetBlizzardRawUriResponse(get_connected_realm_form, hr, region, getNamespace(dynamic_ns, region), &connected_realm_detail)
		if crErr != nil {
			return realmNames
		}

		for _, rlm := range connected_realm_detail.Realms {
			realmNames = append(realmNames, rlm.Name)
		}
	}

	cache_provider.CacheSet(ALL_REALM_NAMES_CACHE, all_realm_key, realmNames, cache_provider.GetDynamicTimeWithShift())

	return realmNames
}

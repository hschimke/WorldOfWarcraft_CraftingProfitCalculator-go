package blizzard_api_helpers

import (
	"fmt"
	"strings"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

const (
	exclude_before_shadowlands           bool   = false
	static_ns                            string = "static"
	dynamic_ns                           string = "dynamic"
	locale_us                            string = "en_US"
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

func checkPageSearchResults(page BlizzardApi.ItemSearch, item_name globalTypes.ItemName) (found_item_id globalTypes.ItemID) {
	found_item_id = 0
	for _, result := range page.Results {
		if strings.EqualFold(item_name, result.Data.Name[locale_us]) {
			found_item_id = result.Data.Id
		}
	}
	return
}

func GetItemId(region globalTypes.RegionCode, item_name globalTypes.ItemName) (globalTypes.ItemID, error) {
	if found, err := cache_provider.CacheCheck(ITEM_SEARCH_CACHE, item_name); err == nil && found {
		item := globalTypes.ItemID(0)
		fndErr := cache_provider.CacheGet(ITEM_SEARCH_CACHE, item_name, &item)
		return item, fndErr
	}

	item_id := uint(0)

	//current_page := uint(0)
	page_count := uint(0)
	const search_api_uri = "/data/wow/search/item"

	fetchPage := BlizzardApi.ItemSearch{}
	_, err := blizzard_api_call.GetBlizzardAPIResponse(region, searchDataPackage{
		"namespace":  getNamespace(static_ns, region),
		"locale":     locale_us,
		"name.en_US": item_name,
		"orderby":    "id:desc",
		"_pageSize":  "1000",
	}, search_api_uri, &fetchPage)
	//current_page = fetchPage.Page
	if err != nil && fetchPage.PageCount <= 0 {
		return 0, fmt.Errorf("no results for %s", item_name)
	}
	page_count = fetchPage.PageCount
	//return fetchPage, fetchPage.Page, fetchPage.PageCount, err

	cpclog.Debug("Found ", page_count, " pages for item search ", item_name)
	if page_count > 0 {
		page_item_id := checkPageSearchResults(fetchPage, item_name)
		if page_item_id > 0 {
			item_id = page_item_id
		} else {
			for cp := fetchPage.Page; cp <= page_count; cp++ {
				cpclog.Silly("Checking page ", cp, " for ", item_name)
				getPage := BlizzardApi.ItemSearch{}
				_, err := blizzard_api_call.GetBlizzardAPIResponse(region, searchPageDataPackage{
					"namespace":  getNamespace(static_ns, region),
					"locale":     locale_us,
					"name.en_US": item_name,
					"orderby":    "id:desc",
					"_pageSize":  "1000",
					"_page":      fmt.Sprint(cp),
				}, search_api_uri, &getPage)
				if err != nil {
					return 0, err
				}
				page_item_id := checkPageSearchResults(getPage, item_name)
				if page_item_id > 0 {
					item_id = page_item_id
					cpclog.Debug("Found ", item_id, " for ", item_name, " on page ", cp, " of ", page_count)
					break
				}
			}
		}
	} else {
		// We didn't get any results, that's an error
		//await cacheSet(ITEM_SEARCH_CACHE, item_name, -1);
		cpclog.Error("No items match search ", item_name)
		return 0, fmt.Errorf("no items match search %s", item_name)
		//throw (new Error('No Results'));
	}

	cache_provider.CacheSet(ITEM_SEARCH_CACHE, item_name, item_id, cache_provider.GetStaticTimeWithShift())

	return item_id, nil
}

func getAllConnectedRealms(region globalTypes.RegionCode) (BlizzardApi.ConnectedRealmIndex, error) {

	const list_connected_realms_api string = "/data/wow/connected-realm/index"
	list_connected_realms_form := basicDataPackage{
		"namespace": getNamespace(dynamic_ns, region),
		"locale":    locale_us,
	}

	var realm_index BlizzardApi.ConnectedRealmIndex
	_, fetchError := blizzard_api_call.GetBlizzardAPIResponse(region, list_connected_realms_form, list_connected_realms_api, &realm_index)
	if fetchError != nil {
		return BlizzardApi.ConnectedRealmIndex{}, fetchError
	}

	return realm_index, nil
}

func GetConnectedRealmId(server_name globalTypes.RealmName, server_region globalTypes.RegionCode) (globalTypes.ConnectedRealmID, error) {
	connected_realm_key := fmt.Sprintf("%s::%s", server_region, server_name)

	if found, err := cache_provider.CacheCheck(CONNECTED_REALM_ID_CACHE, connected_realm_key); err == nil && found {
		item := globalTypes.ConnectedRealmID(0)
		fndErr := cache_provider.CacheGet(CONNECTED_REALM_ID_CACHE, connected_realm_key, &item)
		return item, fndErr
	}

	get_connected_realm_form := basicDataPackage{
		"namespace": getNamespace(dynamic_ns, server_region),
		"locale":    locale_us,
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
		_, crErr := blizzard_api_call.GetBlizzardRawUriResponse(get_connected_realm_form, hr, server_region, &connected_realm_detail)
		if crErr != nil {
			return globalTypes.ConnectedRealmID(0), crErr
		}

		realm_list := connected_realm_detail.Realms
		found_realm := false
		for _, rlm := range realm_list {
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

	cache_provider.CacheSet(CONNECTED_REALM_ID_CACHE, connected_realm_key, realm_id, cache_provider.GetStaticTimeWithShift())
	cpclog.Infof("Found Connected Realm ID: %d for %s %s", realm_id, server_region, server_name)

	// Return that connected realm ID
	return realm_id, nil
}

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

	recipe_options := globalTypes.CraftingStatus{}
	recipe_options.Craftable = false

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

	profession_result_array := make([]globalTypes.CraftingStatus, 0)

	type iData struct {
		profession_list BlizzardApi.ProfessionsIndex
		prof            globalTypes.CharacterProfession
		region          globalTypes.RegionCode
		item_id         globalTypes.ItemID
		item_detail     BlizzardApi.Item
	}

	type iRet struct {
		ret globalTypes.CraftingStatus
		err error
	}

	inputData := make(chan iData, 3)
	outputData := make(chan iRet, len(character_professions))

	workerFunc := func(input chan iData, output chan iRet) {
		for z := range input {
			z, err := checkProfessionCrafting(z.profession_list, z.prof, z.region, z.item_id, z.item_detail)
			output <- iRet{z, err}
		}
	}

	go workerFunc(inputData, outputData)
	go workerFunc(inputData, outputData)
	go workerFunc(inputData, outputData)

	for _, prof := range character_professions {
		inputData <- iData{profession_list, prof, region, item_id, item_detail}
	}
	close(inputData)

	var errArr []error

	for job := 1; job <= len(character_professions); job++ {
		jBr := <-outputData
		if jBr.err != nil {
			errArr = append(errArr, jBr.err)
		}
		profession_result_array = append(profession_result_array, jBr.ret)
	}

	if len(errArr) != 0 {
		return globalTypes.CraftingStatus{}, errArr[0]
	}

	/*
		for _, prof := range character_professions {
			z, err := checkProfessionCrafting(profession_list, prof, region, item_id, item_detail)
			if err != nil {
				return globalTypes.CraftingStatus{}, err
			}
			profession_result_array = append(profession_result_array, z)
		}*/

	// collate professions
	for _, profession_crafting_check := range profession_result_array {
		recipe_options.Recipes = append(recipe_options.Recipes, profession_crafting_check.Recipes...)          //recipe_options.Recipes.concat(profession_crafting_check.recipes)
		recipe_options.Recipe_ids = append(recipe_options.Recipe_ids, profession_crafting_check.Recipe_ids...) //recipe_options.Recipe_ids.concat(profession_crafting_check.recipe_ids)
		recipe_options.Craftable = recipe_options.Craftable || profession_crafting_check.Craftable
	}

	cache_provider.CacheSet(CRAFTABLE_BY_PROFESSION_SET_CACHE, key, &recipe_options, cache_provider.GetComputedTimeWithShift())
	//{craftable: found_craftable, recipe_id: found_recipe_id, crafting_profession: found_profession};
	return recipe_options, nil
}

func getProfessionId(profession_list BlizzardApi.ProfessionsIndex, profession_name string) (uint, error) {
	var id uint = 0
	for _, item := range profession_list.Professions {
		if item.Name == profession_name {
			id = item.Id
			break
		}
	}
	if id == 0 {
		return 0, fmt.Errorf("could not find profession id for %s", profession_name)
	}
	return id, nil
}

func checkProfessionTierCrafting(skill_tier skilltier, region globalTypes.RegionCode, item_id uint, check_profession_id uint, prof string, item_detail BlizzardApi.Item, profession_recipe_options *globalTypes.CraftingStatus) {
	check_scan_tier := strings.Contains(skill_tier.Name, "Shadowlands")
	if !exclude_before_shadowlands {
		check_scan_tier = true
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

						if !crafty && (strings.Contains(skill_tier.Name, "Enchanting") && (strings.Contains(cat.Name, "Enchantments") || strings.Contains(cat.Name, "Echantments"))) {
							cpclog.Sillyf("Checking if uncraftable item %d is craftable with a synthetic item-recipe connection.", item_detail.Id)
							slot := getSlotName(&cat)
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

						if crafty {
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

func checkProfessionCrafting(profession_list BlizzardApi.ProfessionsIndex, prof globalTypes.CharacterProfession, region globalTypes.RegionCode, item_id globalTypes.ItemID, item_detail BlizzardApi.Item) (globalTypes.CraftingStatus, error) {
	cache_key := fmt.Sprintf("%s:%s:%d", region, prof, item_id)
	if found, err := cache_provider.CacheCheck(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key); err == nil && found {
		item := globalTypes.CraftingStatus{}
		fndErr := cache_provider.CacheGet(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key, &item)
		return item, fndErr
	}
	profession_recipe_options := globalTypes.CraftingStatus{}
	profession_recipe_options.Craftable = false

	check_profession_id, err := getProfessionId(profession_list, prof)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}

	// Get a list of the crafting levels for the professions
	profession_detail, err := GetBlizProfessionDetail(check_profession_id, region)
	if err != nil {
		return globalTypes.CraftingStatus{}, err
	}
	crafting_levels := profession_detail.Skill_tiers

	cpclog.Debug("Scanning profession: ", profession_detail.Name)

	// checkProfessionTierCrafting on each crafting level, concurrently.
	for _, tier := range crafting_levels {
		checkProfessionTierCrafting(tier, region, item_id, check_profession_id, prof, item_detail, &profession_recipe_options)
	}
	//await Promise.all(crafting_levels.map((tier) => {
	//    return checkProfessionTierCrafting(tier, region);
	//}));

	cache_provider.CacheSet(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key, profession_recipe_options, cache_provider.GetComputedTimeWithShift())

	return profession_recipe_options, nil

}

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

	return_ids := make([]globalTypes.ItemID, 0)
	for key := range item_ids {
		return_ids = append(return_ids, key)
	}

	return return_ids
}

func getSlotName(category *BlizzardApi.Category) (raw_slot_name string) {
	var name string = category.Name

	raw_slot_name = name

	if loc_sp := strings.Index(name, "Enchantments"); loc_sp > 0 {
		raw_slot_name = name[:loc_sp-1]
		//raw_slot_name = name.slice(0, name.lastIndexOf('Enchantments') - 1);
	} else if loc_incsp := strings.Index(name, "Echantments"); loc_incsp > 0 {
		raw_slot_name = name[:loc_incsp-1]
		//raw_slot_name = name.slice(0, name.lastIndexOf('Echantments') - 1);
	}

	switch raw_slot_name {
	case "Boot":
		raw_slot_name = "Boots"
	case "Glove":
		raw_slot_name = "Gloves"
	}
	return
}

func BuildCyclicRecipeList(region globalTypes.RegionCode) (globalTypes.SkillTierCyclicLinks, error) {
	profession_list, err := GetBlizProfessionsList(region)
	if err != nil {
		return globalTypes.SkillTierCyclicLinks{}, err
	}

	links := make(SkillTierCyclicLinksBuild, 0)

	profz := make([]BlizzardApi.Profession, 0)
	for _, pro := range profession_list.Professions {
		profDetail, err := GetBlizProfessionDetail(pro.Id, region)
		if err != nil {
			return globalTypes.SkillTierCyclicLinks{}, err
		}
		profz = append(profz, profDetail)
	}

	counter := 0
	profession_counter := 0

	type oData struct {
		ret SkillTierCyclicLinksBuild
		err error
	}

	inputData := make(chan BlizzardApi.Profession, 4)
	outputData := make(chan oData, len(profz))

	workerFunc := func(input chan BlizzardApi.Profession, output chan oData) {
		for prof := range input {
			//var collectLinks SkillTierCyclicLinksBuild
			return_data := oData{}
			last_count := counter
			cpclog.Debug("Scanning profession: ", prof.Name, " for cyclic relationships.")
			profession, err := GetBlizProfessionDetail(prof.Id, region)
			if err != nil {
				return_data.err = err
			} else if profession.Skill_tiers != nil {
				for _, st := range profession.Skill_tiers {
					return_data.ret = append(return_data.ret, buildCyclicLinkforSkillTier(st, profession, region)...)
				}
			}
			cpclog.Debug("Scanned ", counter-last_count, " new recipes.")
			profession_counter++
			output <- return_data
		}
	}

	go workerFunc(inputData, outputData)
	go workerFunc(inputData, outputData)
	go workerFunc(inputData, outputData)

	for _, prof := range profz {
		inputData <- prof
	}
	close(inputData)

	var errors []error
	for job := 1; job <= len(profz); job++ {
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

	/*
		for _, prof := range profz {
			last_count := counter
			cpclog.Debug("Scanning profession: ", prof.Name, " for cyclic relationships.")
			profession, err := GetBlizProfessionDetail(prof.Id, region)
			if err != nil {
				return globalTypes.SkillTierCyclicLinks{}, err
			}
			if profession.Skill_tiers != nil {
				for _, st := range profession.Skill_tiers {
					links = append(links, buildCyclicLinkforSkillTier(st, profession, region)...)
				}
			}
			cpclog.Debug("Scanned ", counter-last_count, " new recipes.")
			profession_counter++
		}
	*/

	cpclog.Debug("Scanned ", counter, " recipes in ", profession_counter, " professions")

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

type uint_set struct {
	internal_map map[uint]bool
}

func (s *uint_set) Has(check uint) bool {
	if s.internal_map == nil {
		s.internal_map = make(map[uint]bool)
	}
	_, present := s.internal_map[check]
	return present
}
func (s *uint_set) Add(value uint) {
	if s.internal_map == nil {
		s.internal_map = make(map[uint]bool)
	}
	s.internal_map[value] = true
}

func uint_slice_has(arr []uint, value uint) (found bool) {
	found = false
	for _, v := range arr {
		if v == value {
			found = true
			return
		}
	}
	return
}

func buildCyclicLinkforSkillTier(skill_tier skilltier, profession BlizzardApi.Profession, region globalTypes.RegionCode) SkillTierCyclicLinksBuild {
	cache_key := fmt.Sprintf("%s::%s::%d", region, skill_tier.Name, profession.Id)

	if found, err := cache_provider.CacheCheck(CYCLIC_LINK_CACHE, cache_key); err == nil && found {
		var item SkillTierCyclicLinksBuild
		cache_provider.CacheGet(CYCLIC_LINK_CACHE, cache_key, &item)
		return item
	}

	cpclog.Debug("Scanning st: ", skill_tier.Name)
	checked_set := uint_set{}
	var found_links SkillTierCyclicLinksBuild
	skill_tier_detail, err := GetBlizSkillTierDetail(profession.Id, skill_tier.Id, region)
	if err != nil {
		return SkillTierCyclicLinksBuild{}
	}
	if skill_tier_detail.Categories != nil {
		for _, sk_category := range skill_tier_detail.Categories {
			for _, sk_recipe := range sk_category.Recipes {
				recipe, err := GetBlizRecipeDetail(sk_recipe.Id, region)
				if err != nil {
					return SkillTierCyclicLinksBuild{}
				}
				if !checked_set.Has(recipe.Id) {
					checked_set.Add(recipe.Id)
					//counter++
					if recipe.Reagents != nil && len(recipe.Reagents) == 1 {
						// Go through them all again
						for _, sk_recheck_category := range skill_tier_detail.Categories {
							for _, sk_recheck_recipe := range sk_recheck_category.Recipes {
								recheck_recipe, err := GetBlizRecipeDetail(sk_recheck_recipe.Id, region)
								if err != nil {
									return SkillTierCyclicLinksBuild{}
								}
								if recheck_recipe.Reagents != nil && len(recheck_recipe.Reagents) == 1 && !checked_set.Has(recheck_recipe.Id) {
									r_ids := getRecipeCraftedItemID(recipe)

									rc_ids := getRecipeCraftedItemID(recheck_recipe)

									if uint_slice_has(r_ids, recheck_recipe.Reagents[0].Reagent.Id) {
										if uint_slice_has(rc_ids, recipe.Reagents[0].Reagent.Id) {
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
	}
	cache_provider.CacheSet(CYCLIC_LINK_CACHE, cache_key, found_links, cache_provider.GetStaticTimeWithShift())
	return found_links
}

func GetCraftingRecipe(recipe_id uint, region globalTypes.RegionCode) (BlizzardApi.Recipe, error) {
	return GetBlizRecipeDetail(recipe_id, region)
}

func getNamespace(ns_type string, region globalTypes.RegionCode) string {
	return fmt.Sprintf("%s-%s", ns_type, strings.ToLower(region))
}

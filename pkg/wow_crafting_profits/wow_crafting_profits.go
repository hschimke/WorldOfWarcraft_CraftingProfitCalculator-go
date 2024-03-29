package wow_crafting_profits

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/text_output_helpers"
	"golang.org/x/exp/slices"
)

type recipeCost struct {
	High, Low, Average, Median float64
}

type WoWCpCRunner struct {
	Helper        *blizzard_api_helpers.BlizzardApiHelper
	staticSources static_sources.StaticSources
	Logger        *cpclog.CpCLog
}

/*
Find the value of an item on the auction house.
Items might be for sale on the auction house and be available from vendors.
The auction house items have complicated bonus types.
*/
func getAHItemPrice(item_id globalTypes.ItemID, auction_house *BlizzardApi.Auctions, bonus_level_required uint) globalTypes.AHItemPriceObject {
	// Find the item and return best, worst, average prices
	auction_high := float64(0)
	auction_low := float64(math.MaxUint)
	auction_average := float64(0)
	auction_counter := uint(0)
	auction_average_accumulator := float64(0)
	auctionMedian := float64(0)

	var medianErr error

	//prices := make([]float64, 0, len(auction_house.Auctions)*3)
	prices := make(map[float64]uint64)

	for _, auction := range auction_house.Auctions {
		if auction.Item.Id == item_id {

			if ((bonus_level_required != 0) && (len(auction.Item.Bonus_lists) > 0 && slices.Contains(auction.Item.Bonus_lists, bonus_level_required))) || (bonus_level_required == 0) {
				var foundPrice float64
				if auction.Buyout != 0 {
					foundPrice = float64(auction.Buyout)
				} else {
					foundPrice = float64(auction.Unit_price)
				}
				if foundPrice > auction_high {
					auction_high = foundPrice
				}
				if foundPrice < auction_low {
					auction_low = foundPrice
				}
				auction_average_accumulator += foundPrice * float64(auction.Quantity)

				if _, priceFound := prices[foundPrice]; !priceFound {
					prices[foundPrice] = 0
				}
				prices[foundPrice] = prices[foundPrice] + uint64(auction.Quantity)

				auction_counter += auction.Quantity
			}
		}
	}

	if auction_counter > 0 {
		auction_average = auction_average_accumulator / float64(auction_counter)
	}

	auctionMedian, medianErr = util.MedianFromMap(prices)
	if medianErr != nil {
		auctionMedian = auction_high
	}

	return globalTypes.AHItemPriceObject{
		High:        auction_high,
		Low:         auction_low,
		Average:     auction_average,
		Median:      auctionMedian,
		Total_sales: auction_counter,
	}
}

/*
Retrieve the value of the item from the vendor price,
items that cannot be bought from
vendors are given a value of -1.
*/
func (cpc *WoWCpCRunner) findNoneAHPrice(item_id globalTypes.ItemID, region globalTypes.RegionCode) (float64, error) {
	// Get the item from blizz and see what the purchase price is
	// The general method is to get the item and see if the description mentions the auction house,
	// if it does then return -1, if it doesn't return the 'purchase_price' options
	item, err := cpc.Helper.GetItemDetails(item_id, region)
	if err != nil {
		return 0, err
	}

	vendor_price := float64(-1)
	if item.Description != "" {
		if strings.Contains(item.Description, "vendor") {
			vendor_price = float64(item.Purchase_price)
		}

		if !(strings.Contains(item.Description, "auction")) {
			vendor_price = float64(item.Purchase_price)
		}
	} else {
		vendor_price = float64(item.Purchase_price)
	}
	if item.Purchase_quantity != 0 {
		vendor_price = vendor_price / float64(item.Purchase_quantity)
	}
	return vendor_price, nil
}

/*
Get a list of bonus item values for a given item.

Finds all of the bonus-list types associated with a given item id,
currently the only way to do that is by pulling an auction house down
and then scanning it. If no bonus lists are found an empty array is
returned.
*/
func (cpc *WoWCpCRunner) getItemBonusLists(item_id globalTypes.ItemID, auction_house *BlizzardApi.Auctions) [][]uint {
	var bonus_lists [][]uint
	var bonus_lists_set [][]uint
	for _, auction := range auction_house.Auctions {
		if auction.Item.Id == item_id {
			if len(auction.Item.Bonus_lists) > 0 {
				bonus_lists = append(bonus_lists, auction.Item.Bonus_lists)
			}
		}
	}
	for _, list := range bonus_lists {
		found := false
		for _, i := range bonus_lists_set {
			if len(i) == len(list) && slices.Equal(i, list) {
				found = true
			}
		}
		if !found {
			bonus_lists_set = append(bonus_lists_set, list)
		}
	}
	cpc.Logger.Debug("Item ", item_id, " has ", len(bonus_lists_set), " bonus lists.")
	return bonus_lists_set
}

/*
Bonus levels correspond to a specific increase in item level over base,
get the item level delta for that bonus id.
*/
func (cpc *WoWCpCRunner) getLvlModifierForBonus(bonus_id uint) int {
	raidbots_bonus_lists_ptr, fetchErr := cpc.staticSources.GetBonuses()
	if fetchErr != nil {
		panic(fetchErr)
	}
	if rbl, present := (*raidbots_bonus_lists_ptr)[fmt.Sprint(bonus_id)]; present {
		return rbl.Level
	}
	return 0
}

/**
 * Analyze the profit potential for constructing or buying an item based on available recipes.
 * @param {!string} region The region in which to search.
 * @param {!string} server The server on which to search, server is used for auction house data and prices.
 * @param {Array<string>} character_professions An array of all the available professions.
 * @param {string|number} item The item id or the item name to analyze.
 * @param {number} qauntity The number of items required.
 * @param {?object} passed_ah If an auction house is already available, pass it in and it will be used.
 */
func (cpc *WoWCpCRunner) performProfitAnalysis(region globalTypes.RegionCode, server globalTypes.RealmName, character_professions []globalTypes.CharacterProfession, item globalTypes.ItemSoftIdentity, qauntity uint, passed_ah *BlizzardApi.Auctions, passedCyclicLinks *globalTypes.SkillTierCyclicLinks) (globalTypes.ProfitAnalysisObject, error) {
	// Check if we have to figure out the item id ourselves
	var item_id uint
	if item.ItemId != 0 {
		item_id = item.ItemId
	} else {
		fnd_id, err := cpc.Helper.GetItemId(region, item.ItemName)
		if (fnd_id <= 0) || err != nil {
			cpc.Logger.Error("No itemId could be found for ", item)
			return globalTypes.ProfitAnalysisObject{}, fmt.Errorf("no itemId could be found for %v -> %v", item, err)
		}
		cpc.Logger.Infof("Found %v for %v", fnd_id, item)
		item_id = fnd_id
	}

	raidbots_bonus_lists_ptr, err := cpc.staticSources.GetBonuses()
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}
	raidbots_bonus_lists := *raidbots_bonus_lists_ptr

	rankings_ptr := cpc.staticSources.GetRankMappings()
	rankings := *rankings_ptr

	item_detail, err := cpc.Helper.GetItemDetails(item_id, region)
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}

	base_ilvl := item_detail.Level

	if passedCyclicLinks == nil {
		cos, err := cpc.Helper.BuildCyclicRecipeList(region, &cpc.staticSources)
		passedCyclicLinks = &cos
		if err != nil {
			return globalTypes.ProfitAnalysisObject{}, err
		}
	}

	craftable_item_swaps := *passedCyclicLinks

	price_obj := globalTypes.ProfitAnalysisObject{
		Item_id:       item_id,
		Item_name:     item_detail.Name,
		Item_quantity: float64(qauntity),
	}

	cpc.Logger.Infof("Analyzing profits potential for %s ( %d )", item_detail.Name, item_id)

	// Get the realm id
	server_id, err := cpc.Helper.GetConnectedRealmId(server, region)
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}

	var auction_house *BlizzardApi.Auctions

	//Get the auction house
	if passed_ah == nil {
		ah, err := cpc.Helper.GetAuctionHouse(server_id, region)
		if err != nil {
			return globalTypes.ProfitAnalysisObject{}, err
		}
		auction_house = &ah
	} else {
		auction_house = passed_ah
	}

	// Get Item AH price
	price_obj.Ah_price = getAHItemPrice(item_id, auction_house, 0)

	item_craftable, err := cpc.Helper.CheckIsCrafting(item_id, character_professions, region, &cpc.staticSources)
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}

	// Get NON AH price
	if !item_craftable.Craftable {
		prc, err := cpc.findNoneAHPrice(item_id, region)
		if err != nil {
			return globalTypes.ProfitAnalysisObject{}, err
		}

		price_obj.Vendor_price = prc
	} else {
		price_obj.Vendor_price = 0
	}

	price_obj.Crafting_status = item_craftable

	// Eventually bonus_lists should be treated as separate items and this should happen first
	// When that's the case we should actually return an entire extra set of price data based on each
	// possible bonus_list. They're actually different items, blizz just tells us they aren't.

	//  price_obj.bonus_lists = Array.from(new Set(await getItemBonusLists(item_id, auction_house)));
	price_obj.Bonus_lists = util.FilterArrayToSetDouble(cpc.getItemBonusLists(item_id, auction_house))
	bonus_link := make(map[uint]uint)
	//bl_flat := filterArrayToSet(flattenArray(price_obj.bonus_lists)).filter((bonus: number) => bonus in raidbots_bonus_lists && 'level' in raidbots_bonus_lists[bonus]));)
	fltn_arr := util.FlattenArray(price_obj.Bonus_lists) //Flatten(price_obj.Bonus_lists)
	bl_flat_hld := util.FilterArrayToSet(fltn_arr)
	var bl_flat []uint
	for _, bonus := range bl_flat_hld {
		bns, rb_b_pres := raidbots_bonus_lists[fmt.Sprint(bonus)]
		if rb_b_pres {
			if bns.Level != 0 {
				bl_flat = append(bl_flat, bonus)
			}
		}
	}
	for _, bonus := range bl_flat {
		mod := cpc.getLvlModifierForBonus(bonus)
		if mod != 0 {
			new_level := uint(int(base_ilvl) + mod)
			bonus_link[new_level] = bonus
			cpc.Logger.Debug("Bonus level ", bonus, " results in crafted ilvl of ", new_level)
		}
	}

	recipe_id_list := item_craftable.Recipe_ids

	if item_craftable.Craftable {
		cpc.Logger.Debug("Item ", item_detail.Name, " (", item_id, ") has ", len(item_craftable.Recipes), " recipes.")
		for _, recipe := range item_craftable.Recipes {
			// Get Reagents
			item_bom, err := cpc.Helper.GetBlizRecipeDetail(recipe.Recipe_id, region)
			if err != nil {
				return globalTypes.ProfitAnalysisObject{}, err
			}

			price_obj.Item_quantity = float64(qauntity) / float64(getRecipeOutputValues(item_bom, &cpc.staticSources).Min)

			// Get prices for BOM
			var bom_prices []globalTypes.ProfitAnalysisObject

			cpc.Logger.Debug("Recipe ", item_bom.Name, " (", recipe.Recipe_id, ") has ", len(item_bom.Reagents), " reagents")

			for _, reagent := range item_bom.Reagents {
				if _, fnd := craftable_item_swaps[reagent.Reagent.Id]; fnd {
					cpc.Logger.Error("Cycles are not fully implemented.", craftable_item_swaps[reagent.Reagent.Id])
					return globalTypes.ProfitAnalysisObject{}, fmt.Errorf("cycles are not supported")
				}
				itm := globalTypes.ItemSoftIdentity{
					ItemId: reagent.Reagent.Id,
				}
				new_analysis, err := cpc.performProfitAnalysis(region, server, character_professions, itm, reagent.Quantity, auction_house, passedCyclicLinks)
				if err != nil {
					return globalTypes.ProfitAnalysisObject{}, err
				}
				bom_prices = append(bom_prices, new_analysis)
			}
			rank_level := uint(0)
			var rank_AH globalTypes.AHItemPriceObject
			if len(recipe_id_list) > 1 {
				//var rank_level uint
				if slices.Contains(recipe_id_list, recipe.Recipe_id) {
					for loc, el := range recipe_id_list {
						if el == recipe.Recipe_id {
							if loc < len(rankings.Rank_mapping) {
								rank_level = rankings.Available_levels[rankings.Rank_mapping[loc]]
							} else {
								rank_level = 0
							}
							break
						}
					}
					//rank_level = rankings.Available_levels[rankings.Rank_mapping[recipe_id_list.indexOf(recipe.Recipe_id)]]
				} else {
					rank_level = 0
				}
				//	               rank_level = recipe_id_list.indexOf(recipe.recipe_id) > -1 ? rankings.available_levels[rankings.rank_mapping[recipe_id_list.indexOf(recipe.recipe_id)]] : 0;
				if bonus_link[rank_level] != 0 {
					cpc.Logger.Debugf(`Looking for AH price for %d for level %d using bonus is %d`, item_id, rank_level, bonus_link[rank_level])
					rank_AH = getAHItemPrice(item_id, auction_house, bonus_link[rank_level])
				} else {
					cpc.Logger.Debugf(`Item %d has no auctions for level %d`, item_id, rank_level)
				}
			}

			price_obj.Recipe_options = append(price_obj.Recipe_options, globalTypes.RecipeOption{
				Recipe:  recipe,
				Prices:  bom_prices,
				Rank:    rank_level,
				Rank_ah: rank_AH,
			})
		}
	} else {
		cpc.Logger.Debugf(`Item %s (%d) not craftable with professions: %v`, item_detail.Name, item_id, character_professions)
		if len(price_obj.Bonus_lists) > 0 {
			//price_obj.bonus_prices = [];
			for _, bonus := range bl_flat {
				rbl := raidbots_bonus_lists[fmt.Sprint(bonus)]
				level_uncrafted_ah_cost := struct {
					Level uint
					Ah    globalTypes.AHItemPriceObject
				}{
					Level: uint(int(base_ilvl) + rbl.Level),
					Ah:    getAHItemPrice(item_id, auction_house, bonus),
				}
				price_obj.Bonus_prices = append(price_obj.Bonus_prices, level_uncrafted_ah_cost)
			}
		}
	}

	return price_obj, nil
}

/**
 * Figure out the best/worst/average cost to construct a recipe given all items required.
 * @param recipe_option The recipe to price.
 */
func (cpc *WoWCpCRunner) recipeCostCalculator(recipe_option globalTypes.RecipeOption) recipeCost {
	/**
	 * For each recipe
	 *   For each component
	 *     if component is vendor: cost = price * quantity
	 *     if component is on AH: cost = h/l/a * quantity (tuple)
	 *     if component is craftable: cost = h/l/a of each recipe option
	 */
	var cost recipeCost

	for _, component := range recipe_option.Prices {
		if component.Vendor_price > 0 {
			cost.High += component.Vendor_price * component.Item_quantity
			cost.Low += component.Vendor_price * component.Item_quantity
			cost.Average += component.Vendor_price * component.Item_quantity
			cost.Median += component.Vendor_price * component.Item_quantity
			cpc.Logger.Debug("Use vendor price for ", component.Item_name, " (", component.Item_id, ")")
		} else if !component.Crafting_status.Craftable {

			high := float64(0)
			low := float64(math.MaxUint64)
			average := float64(0)
			count := 0
			if component.Ah_price.Total_sales > 0 {
				average += component.Ah_price.Average
				if component.Ah_price.High > high {
					high = component.Ah_price.High
				}
				if component.Ah_price.Low < low {
					low = component.Ah_price.Low
				}
				count++
			}
			cost.Average += (average / float64(count)) * float64(component.Item_quantity)
			cost.High += high * component.Item_quantity
			cost.Low += low * component.Item_quantity
			cost.Median += component.Ah_price.Median * component.Item_quantity
			cpc.Logger.Debugf("Use auction price for uncraftable item %s (%d)", component.Item_name, component.Item_id)
		} else {
			cpc.Logger.Debugf("Recursive check for item %s (%d)", component.Item_name, component.Item_id)
			ave_acc := float64(0)
			ave_cnt := 0

			high := float64(0)
			low := math.MaxFloat64

			var costs []float64

			for _, opt := range component.Recipe_options {
				recurse_price := cpc.recipeCostCalculator(opt)

				if high < recurse_price.High*component.Item_quantity {
					high = recurse_price.High * component.Item_quantity
				}

				if low > recurse_price.Low*component.Item_quantity {
					low = recurse_price.Low * component.Item_quantity
				}

				costs = append(costs, recurse_price.Median*component.Item_quantity)

				ave_acc += recurse_price.Average * float64(component.Item_quantity)
				ave_cnt++
			}

			cost.Low = low
			cost.High = high
			cost.Average += ave_acc / float64(ave_cnt)
			if med, medErr := util.Median(costs); medErr != nil {
				cost.Median = high
			} else {
				cost.Median = med
			}
		}
	}

	if cost.Average == 0 || math.IsNaN(cost.Average) {
		cost.High = 0
		cost.Low = 0
		cost.Median = 0
		cost.Average = 0
	}

	return cost
}

/**
 * Create an object used for constructing shopping lists and formatted output data.
 * @param {!object} price_data The object created by the analyze function.
 * @param {!string} region The region in which to work.
 */
func (cpc *WoWCpCRunner) generateOutputFormat(price_data globalTypes.ProfitAnalysisObject, region globalTypes.RegionCode) globalTypes.OutputFormatObject {
	object_output := globalTypes.OutputFormatObject{
		Name:     price_data.Item_name,
		Id:       price_data.Item_id,
		Required: price_data.Item_quantity,
		Recipes:  make([]globalTypes.OutputFormatRecipe, 0),
	}

	if (price_data.Ah_price.Total_sales != 0) && (price_data.Ah_price.Total_sales > 0) {
		object_output.Ah = globalTypes.OutputFormatPrice{
			Sales:   price_data.Ah_price.Total_sales,
			High:    price_data.Ah_price.High,
			Low:     price_data.Ah_price.Low,
			Average: price_data.Ah_price.Average,
			Median:  price_data.Ah_price.Median,
		}
	}
	if price_data.Vendor_price > 0 {
		object_output.Vendor = price_data.Vendor_price
	}

	for _, recipe_option := range price_data.Recipe_options {
		option_price := cpc.recipeCostCalculator(recipe_option)
		recipe, err := cpc.Helper.GetBlizRecipeDetail(recipe_option.Recipe.Recipe_id, region)
		if err != nil {
			return globalTypes.OutputFormatObject{}
		}
		obj_recipe := globalTypes.OutputFormatRecipe{
			Name:    recipe.Name,
			Rank:    recipe_option.Rank,
			Id:      recipe_option.Recipe.Recipe_id,
			Output:  getRecipeOutputValues(recipe, &cpc.staticSources),
			High:    option_price.High,
			Low:     option_price.Low,
			Average: option_price.Average,
			Median:  option_price.Median,
		}
		//obj_recipe.parts = [];

		if (recipe_option.Rank_ah.Total_sales != 0) && (recipe_option.Rank_ah.Total_sales > 0) {
			obj_recipe.Ah = globalTypes.OutputFormatPrice{
				Sales:   recipe_option.Rank_ah.Total_sales,
				High:    recipe_option.Rank_ah.High,
				Low:     recipe_option.Rank_ah.Low,
				Average: recipe_option.Rank_ah.Average,
				Median:  recipe_option.Rank_ah.Median,
			}
		}
		//let prom_list = [];

		for _, opt := range recipe_option.Prices {
			obj_recipe.Parts = append(obj_recipe.Parts, cpc.generateOutputFormat(opt, region))
		}

		object_output.Recipes = append(object_output.Recipes, obj_recipe)
	}

	for _, bonus_price := range price_data.Bonus_prices {
		object_output.Bonus_prices = append(object_output.Bonus_prices, globalTypes.OutputFormatBonusPrices{
			Level: bonus_price.Level,
			Ah: globalTypes.OutputFormatPrice{
				Sales:   bonus_price.Ah.Total_sales,
				High:    bonus_price.Ah.High,
				Low:     bonus_price.Ah.Low,
				Average: bonus_price.Ah.Average,
				Median:  bonus_price.Ah.Median,
			}})
	}

	return object_output
}

func getRecipeOutputValues(recipe BlizzardApi.Recipe, static_source *static_sources.StaticSources) globalTypes.OutpoutFormatRecipeOutput {
	var min, max, value float64

	safe_found := false
	if recipe.Crafted_quantity.Minimum != 0 {
		min = recipe.Crafted_quantity.Minimum
		safe_found = true
	}
	if recipe.Crafted_quantity.Maximum != 0 {
		max = recipe.Crafted_quantity.Maximum
		safe_found = true
	}
	if recipe.Crafted_quantity.Value != 0 {
		value = recipe.Crafted_quantity.Value
		safe_found = true
	}

	if !safe_found {
		firesong_list, firesong_list_err := static_source.GetFireSongsCraftingLinkTable()
		if firesong_list_err == nil {
			for _, element := range *firesong_list {
				if element.RecipeId == recipe.Id {
					value = 1
				}
			}
		}
	}

	if min == 0 && max == 0 {
		min = value
		max = value
	}

	return globalTypes.OutpoutFormatRecipeOutput{
		Min:   min,
		Max:   max,
		Value: value,
	}
}

/**
 * Return the ranks available for the top level item generated from generateOutputFormat.
 * @param {!object} intermediate_data Data from generateOutputFormat.
 */
func getShoppingListRanks(intermediate_data globalTypes.OutputFormatObject) []uint {
	ranks := make([]uint, 0)
	for _, recipe := range intermediate_data.Recipes {
		ranks = append(ranks, recipe.Rank)
	}
	return ranks
}

/**
 * Construct a shopping list given a provided inventory object.
 * @param {!object} intermediate_data Data from generateOutputFormat.
 * @param {!RunConfiguration} on_hand A provided inventory to get existing items from.
 */
func (cpc *WoWCpCRunner) constructShoppingList(intermediate_data globalTypes.OutputFormatObject, on_hand *globalTypes.RunConfiguration) globalTypes.OutputFormatShoppingList {
	shopping_lists := make(globalTypes.OutputFormatShoppingList)
	for _, rank := range getShoppingListRanks(intermediate_data) {
		cpc.Logger.Debug("Resetting inventory for rank shopping list.")
		on_hand.ResetInventoryAdjustments()
		shopping_list := cpc.build_shopping_list(intermediate_data, rank)
		for listIndex, li := range shopping_list {
			needed := li.Quantity
			available := on_hand.ItemCount(li.Id)

			cpc.Logger.Debugf("%s (%d) %f needed with %d available", li.Name, li.Id, needed, available)
			if needed <= float64(available) {
				cpc.Logger.Debugf("$%s (%d) used %f of the available %d", li.Name, li.Id, needed, available)
				needed = 0
				on_hand.AdjustInventory(li.Id, (int(needed) * -1))
			} else if (needed > float64(available)) && (int(available) != 0) {
				needed -= float64(available)
				cpc.Logger.Debugf("%s (%d) used all of the available %d and still need %f", li.Name, li.Id, available, needed)
				on_hand.AdjustInventory(li.Id, (int(available) * -1))
			}

			li.Quantity = needed

			// Update the cost for this list item
			if li.Cost.Vendor != 0 {
				li.Cost.Vendor *= li.Quantity
			}
			if li.Cost.Ah.Sales != 0 {
				li.Cost.Ah.High *= li.Quantity
				li.Cost.Ah.Low *= li.Quantity
				li.Cost.Ah.Median *= li.Quantity
				li.Cost.Ah.Average *= float64(li.Quantity)
			}

			shopping_list[listIndex] = li
		}
		shopping_lists[rank] = shopping_list
	}
	return shopping_lists
}

/**
 * Build a raw shopping list using generateOutputFormat data, ignores inventory information.
 * @param {!object} intermediate_data The generateOutputFormat data used for construction.
 * @param {number} rank_requested The specific rank to generate a list for, only matters for legendary base items in Shadowlands.
 */
func (cpc *WoWCpCRunner) build_shopping_list(intermediate_data globalTypes.OutputFormatObject, rank_requested uint) []globalTypes.ShoppingList {
	shopping_list := make([]globalTypes.ShoppingList, 0)

	shopping_recipe_exclusions_ptr := cpc.staticSources.GetShoppingRecipeExclusionList()
	shopping_recipe_exclusions := *shopping_recipe_exclusions_ptr

	cpc.Logger.Debugf(`Build shopping list for %s (%d) rank %d`, intermediate_data.Name, intermediate_data.Id, rank_requested)

	needed := intermediate_data.Required

	if len(intermediate_data.Recipes) == 0 {
		shopping_list = append(shopping_list, globalTypes.ShoppingList{
			Id:       intermediate_data.Id,
			Name:     intermediate_data.Name,
			Quantity: intermediate_data.Required,
			Cost: globalTypes.ShoppingListCost{
				Ah:     intermediate_data.Ah,
				Vendor: intermediate_data.Vendor,
			},
		})
		cpc.Logger.Debug(intermediate_data.Name, "(", intermediate_data.Id, ") cannot be crafted.")
	} else {
		for _, recipe := range intermediate_data.Recipes {
			// Make sure the recipe isn't on the exclusion list
			if slices.Contains(shopping_recipe_exclusions.Exclusions, recipe.Id) {
				cpc.Logger.Debug(recipe.Name, " (", recipe.Id, ") is on the exclusion list. Add it directly")
				shopping_list = append(shopping_list, globalTypes.ShoppingList{
					Id:       intermediate_data.Id,
					Name:     intermediate_data.Name,
					Quantity: intermediate_data.Required,
					Cost: globalTypes.ShoppingListCost{
						Ah:     intermediate_data.Ah,
						Vendor: intermediate_data.Vendor,
					},
				})
			} else {
				if recipe.Rank == rank_requested {
					for _, part := range recipe.Parts {
						// Only top level searches can have ranks
						for _, sl := range cpc.build_shopping_list(part, 0) {
							//let al = sl;
							cpc.Logger.Debugf(`Need %f of %s (%d) for each of %f`, sl.Quantity, sl.Name, sl.Id, needed)

							sl.Quantity = sl.Quantity * needed
							shopping_list = append(shopping_list, sl)
						}
					}
				} else {
					cpc.Logger.Debugf(`Skipping recipe %d because its rank (%d) does not match the requested rank (%d)`, recipe.Id, recipe.Rank, rank_requested)
				}
			}
		}
	}

	// Build the return shopping list.
	tmp := make(map[uint]globalTypes.ShoppingList)
	ret_list := make([]globalTypes.ShoppingList, 0)
	//logger.debug(shopping_list);
	for _, list_element := range shopping_list {
		hld, present := tmp[list_element.Id]
		if !present {
			hld.Id = list_element.Id
			hld.Name = list_element.Name
			hld.Quantity = 0
			hld.Cost = list_element.Cost
		}
		hld.Quantity += list_element.Quantity
		tmp[list_element.Id] = hld
	}

	for _, list := range tmp {
		ret_list = append(ret_list, list)
	}

	return ret_list
}

// Get the globalTypes.RegionCode version of a string or an error
func getRegionCode(region string) (region_coded globalTypes.RegionCode, err error) {
	check_str := strings.ToLower(region)
	err = nil
	switch check_str {
	case "us":
		region_coded = globalTypes.RegionCode(check_str)
	case "eu":
		region_coded = globalTypes.RegionCode(check_str)
	case "kr":
		region_coded = globalTypes.RegionCode(check_str)
	case "tw":
		region_coded = globalTypes.RegionCode(check_str)
	default:
		err = fmt.Errorf("%s is invalid. Valid regions include 'us', 'eu', 'kr', and 'tw'", region)
	}
	return
}

/**
 * Perform a full run of the profit analyzer, beginning with profit analyze and finishing with various output formats.
 *
 * @param {!string} region The region in which to search.
 * @param {!server} server The server on which the profits should be calculated.
 * @param {!Array<string>} professions An array of available professions.
 * @param {!string|number} item The item id or name to analyze.
 * @param {!RunConfiguration} json_config A RunConfiguration object containing the available inventory.
 * @param {!number} count The number of items required.
 */
func (cpc *WoWCpCRunner) run(region string, server globalTypes.RealmName, useAllProfessions bool, professions_input []globalTypes.CharacterProfession, item globalTypes.ItemSoftIdentity, json_config *globalTypes.RunConfiguration, count uint) (globalTypes.RunReturn, error) {

	cpc.Logger.Info("World of Warcraft Crafting Profit Calculator")

	cpc.Logger.Infof("Checking %s in %s for %v with available professions %s", server, region, item, professions_input)

	//let formatted_data = 'NO DATA';

	encoded_region, err := getRegionCode(region)
	if err != nil {
		return globalTypes.RunReturn{Formatted: "NO DATA"}, err
	}

	var professions []globalTypes.CharacterProfession

	if useAllProfessions {
		profList, profErr := cpc.Helper.GetBlizProfessionsList(region)
		if profErr != nil {
			return globalTypes.RunReturn{Formatted: "NO DATA"}, profErr
		}
		for _, prof := range profList.Professions {
			professions = append(professions, prof.Name)
		}
	} else {
		professions = professions_input
	}

	price_data, err := cpc.performProfitAnalysis(encoded_region, server, professions, item, count, nil, nil)
	if err != nil {
		return globalTypes.RunReturn{Formatted: "NO DATA"}, err
	}
	intermediate_data := cpc.generateOutputFormat(price_data, encoded_region)
	intermediate_data.Shopping_lists = cpc.constructShoppingList(intermediate_data, json_config)
	formatted_data := text_output_helpers.TextFriendlyOutputFormat(&intermediate_data, 0)

	return globalTypes.RunReturn{
		Price:        price_data,
		Intermediate: intermediate_data,
		Formatted:    formatted_data,
	}, nil
}

/**
 * Save the generated output to the filesystem.
 * @param price_data The price data.
 * @param intermediate_data The output cost object with shopping list.
 * @param formatted_data The preformatted text output with shopping list.
 */
func saveOutput(price_data globalTypes.ProfitAnalysisObject, intermediate_data globalTypes.OutputFormatObject, formatted_data string, logger *cpclog.CpCLog) error {
	const (
		intermediate_output_fn string = "intermediate_output.json"
		formatted_output_fn    string = "formatted_output"
		raw_output_fn          string = "raw_output.json"
	)

	logger.Info("Saving output")
	if intermediate_data.Id != 0 {
		intFile, err := os.Create(intermediate_output_fn)
		if err != nil {
			return err
		}
		defer intFile.Close()
		encoder := json.NewEncoder(intFile)
		encoder.SetIndent("", "  ")
		encode_err := encoder.Encode(&intermediate_data)
		if encode_err != nil {
			fmt.Print(encode_err.Error())
			return encode_err
		}
		logger.Info("Intermediate output saved")
	}
	forFile, err := os.Create(formatted_output_fn)
	if err != nil {
		return err
	}
	defer forFile.Close()

	formatted_writer := bufio.NewWriter(forFile)
	defer formatted_writer.Flush()

	_, writer_err := formatted_writer.WriteString(formatted_data)
	if writer_err != nil {
		logger.Error("Issue writing to file for formatted data: ", writer_err)
	}

	logger.Info("Formatted output saved")

	if price_data.Item_id != 0 {
		rawFile, err := os.Create(raw_output_fn)
		if err != nil {
			return err
		}
		defer rawFile.Close()
		encoder := json.NewEncoder(rawFile)
		encoder.SetIndent("", "  ")
		encode_err := encoder.Encode(&price_data)
		if encode_err != nil {
			return encode_err
		}
		logger.Info("Raw output saved")
	}
	return nil
}

/**
 * Perform a run with pure json configuration from the addon.
 * @param {RunConfiguration} json_config The configuration object.
 */
func (cpc *WoWCpCRunner) RunWithJSONConfig(json_config *globalTypes.RunConfiguration) (globalTypes.RunReturn, error) {
	return cpc.run(json_config.Realm_region, json_config.Realm_name, json_config.UseAllProfessions, json_config.Professions, json_config.Item, json_config, json_config.Item_count)
	//return globalTypes.RunReturn{}, fmt.Errorf("not implemented")
}

/**
 * Run from the command prompt.
 * @param {RunConfiguration} json_config The configuration object to execute.
 */
func (cpc *WoWCpCRunner) CliRun(json_config *globalTypes.RunConfiguration) error {
	results, err := cpc.RunWithJSONConfig(json_config)
	if err != nil {
		return err
	}
	saveOutput(results.Price, results.Intermediate, results.Formatted, cpc.Logger)
	return nil
}

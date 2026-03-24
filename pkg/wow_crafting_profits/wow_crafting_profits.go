package wow_crafting_profits

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/text_output_helpers"
)

type recipeCost struct {
	High, Low, Average, Median float64
}

type WoWCpCRunner struct {
	Helper          *blizzard_api_helpers.BlizzardApiHelper
	staticSources   static_sources.StaticSources
	Logger          *cpclog.CpCLog
	indexedAuctions map[globalTypes.ItemID][]BlizzardApi.Auction
}

func (cpc *WoWCpCRunner) indexAuctions(auction_house *BlizzardApi.Auctions) {
	cpc.indexedAuctions = make(map[globalTypes.ItemID][]BlizzardApi.Auction)
	for _, auction := range auction_house.Auctions {
		cpc.indexedAuctions[auction.Item.Id] = append(cpc.indexedAuctions[auction.Item.Id], auction)
	}
}

/*
Find the value of an item on the auction house.
Items might be for sale on the auction house and be available from vendors.
The auction house items have complicated bonus types.
*/
func getAHItemPrice(item_id globalTypes.ItemID, indexedAuctions map[globalTypes.ItemID][]BlizzardApi.Auction, bonus_level_required uint) globalTypes.AHItemPriceObject {
	// Find the item and return best, worst, average prices
	auction_high := float64(0)
	auction_low := float64(math.MaxUint)
	auction_average := float64(0)
	auction_counter := uint(0)
	auction_average_accumulator := float64(0)
	auctionMedian := float64(0)

	var medianErr error

	prices := make(map[float64]uint64)

	auctions, present := indexedAuctions[item_id]
	if !present {
		return globalTypes.AHItemPriceObject{
			Low: auction_low,
		}
	}

	for _, auction := range auctions {
		// Filter by bonus level if required.
		// In modern WoW, we might also want to filter by quality or modifiers,
		// but for now we'll stick to the bonus_level_required logic which covers many cases.
		if ((bonus_level_required != 0) && (len(auction.Item.Bonus_lists) > 0 && slices.Contains(auction.Item.Bonus_lists, bonus_level_required))) || (bonus_level_required == 0) {
			var foundPrice float64
			// In modern API:
			// Regular auctions have 'buyout' or 'bid'.
			// Commodities have 'unit_price'.
			if auction.Unit_price != 0 {
				foundPrice = float64(auction.Unit_price)
			} else if auction.Buyout != 0 {
				foundPrice = float64(auction.Buyout)
			} else if auction.Bid != 0 {
				foundPrice = float64(auction.Bid)
			}

			if foundPrice == 0 {
				continue
			}

			if foundPrice > auction_high {
				auction_high = foundPrice
			}
			if foundPrice < auction_low {
				auction_low = foundPrice
			}
			auction_average_accumulator += foundPrice * float64(auction.Quantity)

			prices[foundPrice] += uint64(auction.Quantity)
			auction_counter += auction.Quantity
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
func (cpc *WoWCpCRunner) findNoneAHPrice(ctx context.Context, item_id globalTypes.ItemID, region globalTypes.RegionCode) (float64, error) {
	// Get the item from blizz and see what the purchase price is
	item, err := cpc.Helper.GetItemDetails(ctx, item_id, region)
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
*/
func (cpc *WoWCpCRunner) getItemBonusLists(item_id globalTypes.ItemID) [][]uint {
	var bonus_lists [][]uint
	if auctions, present := cpc.indexedAuctions[item_id]; present {
		for _, auction := range auctions {
			if len(auction.Item.Bonus_lists) > 0 {
				bonus_lists = append(bonus_lists, auction.Item.Bonus_lists)
			}
		}
	}
	bonus_lists_set := util.FilterArrayToSetDouble(bonus_lists)
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
		return 0
	}
	if rbl, present := (*raidbots_bonus_lists_ptr)[fmt.Sprint(bonus_id)]; present {
		return rbl.Level
	}
	return 0
}

/**
 * Analyze the profit potential for constructing or buying an item based on available recipes.
 */
func (cpc *WoWCpCRunner) performProfitAnalysis(ctx context.Context, region globalTypes.RegionCode, server globalTypes.RealmName, character_professions []globalTypes.CharacterProfession, item globalTypes.ItemSoftIdentity, qauntity uint, passed_ah *BlizzardApi.Auctions, passedCyclicLinks *globalTypes.SkillTierCyclicLinks) (globalTypes.ProfitAnalysisObject, error) {
	// Check if we have to figure out the item id ourselves
	var item_id uint
	if item.ItemId != 0 {
		item_id = item.ItemId
	} else {
		fnd_id, err := cpc.Helper.GetItemId(ctx, region, item.ItemName)
		if (fnd_id <= 0) || err != nil {
			cpc.Logger.Error("No itemId could be found for ", item)
			return globalTypes.ProfitAnalysisObject{}, fmt.Errorf("no itemId could be found for %v -> %v", item, err)
		}
		cpc.Logger.Infof("Found %v for %v", fnd_id, item)
		item_id = uint(fnd_id)
	}

	raidbots_bonus_lists_ptr, err := cpc.staticSources.GetBonuses()
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}
	raidbots_bonus_lists := *raidbots_bonus_lists_ptr

	rankings_ptr := cpc.staticSources.GetRankMappings()
	rankings := *rankings_ptr

	item_detail, err := cpc.Helper.GetItemDetails(ctx, globalTypes.ItemID(item_id), region)
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}

	base_ilvl := item_detail.Level

	if passedCyclicLinks == nil {
		cos, err := cpc.Helper.BuildCyclicRecipeList(ctx, region, &cpc.staticSources)
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
	server_id, err := cpc.Helper.GetConnectedRealmId(ctx, server, region)
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}

	var auction_house *BlizzardApi.Auctions

	//Get the auction house
	if passed_ah == nil {
		ah, err := cpc.Helper.GetAuctionHouse(ctx, server_id, region)
		if err != nil {
			return globalTypes.ProfitAnalysisObject{}, err
		}
		auction_house = &ah
	} else {
		auction_house = passed_ah
	}

	if cpc.indexedAuctions == nil {
		cpc.indexAuctions(auction_house)
	}

	// Get Item AH price
	price_obj.Ah_price = getAHItemPrice(globalTypes.ItemID(item_id), cpc.indexedAuctions, 0)

	item_craftable, err := cpc.Helper.CheckIsCrafting(ctx, globalTypes.ItemID(item_id), character_professions, region, &cpc.staticSources)
	if err != nil {
		return globalTypes.ProfitAnalysisObject{}, err
	}

	// Get NON AH price
	if !item_craftable.Craftable {
		prc, err := cpc.findNoneAHPrice(ctx, globalTypes.ItemID(item_id), region)
		if err != nil {
			return globalTypes.ProfitAnalysisObject{}, err
		}

		price_obj.Vendor_price = prc
	} else {
		price_obj.Vendor_price = 0
	}

	price_obj.Crafting_status = item_craftable

	price_obj.Bonus_lists = util.FilterArrayToSetDouble(cpc.getItemBonusLists(globalTypes.ItemID(item_id)))
	bonus_link := make(map[uint]uint)
	fltn_arr := util.FlattenArray(price_obj.Bonus_lists)
	bl_flat_hld := util.FilterArrayToSet(fltn_arr)
	var bl_flat []uint
	for _, bonus := range bl_flat_hld {
		if bns, rb_b_pres := raidbots_bonus_lists[fmt.Sprint(bonus)]; rb_b_pres && bns.Level != 0 {
			bl_flat = append(bl_flat, bonus)
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
		
		// Use errgroup for parallel recipe analysis
		g, gCtx := errgroup.WithContext(ctx)
		recipeOptions := make([]globalTypes.RecipeOption, len(item_craftable.Recipes))
		var mu sync.Mutex

		for i, recipe := range item_craftable.Recipes {
			i, recipe := i, recipe
			g.Go(func() error {
				item_bom, err := cpc.Helper.GetBlizRecipeDetail(gCtx, recipe.Recipe_id, region)
				if err != nil {
					return err
				}

				// Reagent prices analysis
				bom_prices := make([]globalTypes.ProfitAnalysisObject, len(item_bom.Reagents))
				rg, rgCtx := errgroup.WithContext(gCtx)

				for j, reagent := range item_bom.Reagents {
					j, reagent := j, reagent
					if _, fnd := craftable_item_swaps[reagent.Reagent.Id]; fnd {
						return fmt.Errorf("cycles are not supported for reagent %d", reagent.Reagent.Id)
					}
					rg.Go(func() error {
						itm := globalTypes.ItemSoftIdentity{ItemId: reagent.Reagent.Id}
						new_analysis, err := cpc.performProfitAnalysis(rgCtx, region, server, character_professions, itm, reagent.Quantity, auction_house, passedCyclicLinks)
						if err != nil {
							return err
						}
						bom_prices[j] = new_analysis
						return nil
					})
				}
				if err := rg.Wait(); err != nil {
					return err
				}

				rank_level := uint(0)
				var rank_AH globalTypes.AHItemPriceObject
				if len(recipe_id_list) > 1 {
					if loc := slices.Index(recipe_id_list, recipe.Recipe_id); loc != -1 {
						if loc < len(rankings.Rank_mapping) {
							rank_level = rankings.Available_levels[rankings.Rank_mapping[loc]]
						}
					}
					if bonus_link[rank_level] != 0 {
						rank_AH = getAHItemPrice(globalTypes.ItemID(item_id), cpc.indexedAuctions, bonus_link[rank_level])
					}
				}

				mu.Lock()
				recipeOptions[i] = globalTypes.RecipeOption{
					Recipe:  recipe,
					Prices:  bom_prices,
					Rank:    rank_level,
					Rank_ah: rank_AH,
				}
				mu.Unlock()
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return globalTypes.ProfitAnalysisObject{}, err
		}
		price_obj.Recipe_options = recipeOptions

	} else {
		cpc.Logger.Debugf(`Item %s (%d) not craftable with professions: %v`, item_detail.Name, item_id, character_professions)
		if len(price_obj.Bonus_lists) > 0 {
			price_obj.Bonus_prices = make([]struct {
				Level uint
				Ah    globalTypes.AHItemPriceObject
			}, 0, len(bl_flat))
			for _, bonus := range bl_flat {
				rbl := raidbots_bonus_lists[fmt.Sprint(bonus)]
				level_uncrafted_ah_cost := struct {
					Level uint
					Ah    globalTypes.AHItemPriceObject
				}{
					Level: uint(int(base_ilvl) + rbl.Level),
					Ah:    getAHItemPrice(globalTypes.ItemID(item_id), cpc.indexedAuctions, bonus),
				}
				price_obj.Bonus_prices = append(price_obj.Bonus_prices, level_uncrafted_ah_cost)
			}
		}
	}

	return price_obj, nil
}

func (cpc *WoWCpCRunner) recipeCostCalculator(recipe_option globalTypes.RecipeOption) recipeCost {
	var cost recipeCost

	for _, component := range recipe_option.Prices {
		if component.Vendor_price > 0 {
			cost.High += component.Vendor_price * component.Item_quantity
			cost.Low += component.Vendor_price * component.Item_quantity
			cost.Average += component.Vendor_price * component.Item_quantity
			cost.Median += component.Vendor_price * component.Item_quantity
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
			if count > 0 {
				cost.Average += (average / float64(count)) * float64(component.Item_quantity)
			}
			cost.High += high * component.Item_quantity
			cost.Low += low * component.Item_quantity
			cost.Median += component.Ah_price.Median * component.Item_quantity
		} else {
			ave_acc := float64(0)
			ave_cnt := 0
			high := float64(0)
			low := math.MaxFloat64
			costs := make([]float64, 0, len(component.Recipe_options))

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
			if ave_cnt > 0 {
				cost.Average += ave_acc / float64(ave_cnt)
			}
			if med, medErr := util.Median(costs); medErr != nil {
				cost.Median = high
			} else {
				cost.Median = med
			}
		}
	}

	if math.IsNaN(cost.Average) {
		cost.Average = 0
	}

	return cost
}

func (cpc *WoWCpCRunner) generateOutputFormat(ctx context.Context, price_data globalTypes.ProfitAnalysisObject, region globalTypes.RegionCode) globalTypes.OutputFormatObject {
	object_output := globalTypes.OutputFormatObject{
		Name:         price_data.Item_name,
		Id:           price_data.Item_id,
		Required:     price_data.Item_quantity,
		Recipes:      make([]globalTypes.OutputFormatRecipe, 0, len(price_data.Recipe_options)),
		Bonus_prices: make([]globalTypes.OutputFormatBonusPrices, 0, len(price_data.Bonus_prices)),
	}

	if price_data.Ah_price.Total_sales > 0 {
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
		recipe, err := cpc.Helper.GetBlizRecipeDetail(ctx, recipe_option.Recipe.Recipe_id, region)
		if err != nil {
			continue
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
			Parts:   make([]globalTypes.OutputFormatObject, 0, len(recipe_option.Prices)),
		}

		if recipe_option.Rank_ah.Total_sales > 0 {
			obj_recipe.Ah = globalTypes.OutputFormatPrice{
				Sales:   recipe_option.Rank_ah.Total_sales,
				High:    recipe_option.Rank_ah.High,
				Low:     recipe_option.Rank_ah.Low,
				Average: recipe_option.Rank_ah.Average,
				Median:  recipe_option.Rank_ah.Median,
			}
		}

		for _, opt := range recipe_option.Prices {
			obj_recipe.Parts = append(obj_recipe.Parts, cpc.generateOutputFormat(ctx, opt, region))
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

func getShoppingListRanks(intermediate_data globalTypes.OutputFormatObject) []uint {
	ranks := make([]uint, 0, len(intermediate_data.Recipes))
	for _, recipe := range intermediate_data.Recipes {
		ranks = append(ranks, recipe.Rank)
	}
	return ranks
}

func (cpc *WoWCpCRunner) constructShoppingList(intermediate_data globalTypes.OutputFormatObject, on_hand *globalTypes.RunConfiguration) globalTypes.OutputFormatShoppingList {
	shopping_lists := make(globalTypes.OutputFormatShoppingList)
	for _, rank := range getShoppingListRanks(intermediate_data) {
		on_hand.ResetInventoryAdjustments()
		shopping_list := cpc.build_shopping_list(intermediate_data, rank)
		for listIndex, li := range shopping_list {
			needed := li.Quantity
			available := on_hand.ItemCount(li.Id)

			if needed <= float64(available) {
				needed = 0
				on_hand.AdjustInventory(li.Id, (int(needed) * -1))
			} else if (needed > float64(available)) && (int(available) != 0) {
				needed -= float64(available)
				on_hand.AdjustInventory(li.Id, (int(available) * -1))
			}

			li.Quantity = needed

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

func (cpc *WoWCpCRunner) build_shopping_list(intermediate_data globalTypes.OutputFormatObject, rank_requested uint) []globalTypes.ShoppingList {
	shopping_list := make([]globalTypes.ShoppingList, 0)

	shopping_recipe_exclusions_ptr := cpc.staticSources.GetShoppingRecipeExclusionList()
	shopping_recipe_exclusions := *shopping_recipe_exclusions_ptr

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
	} else {
		for _, recipe := range intermediate_data.Recipes {
			if slices.Contains(shopping_recipe_exclusions.Exclusions, recipe.Id) {
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
						for _, sl := range cpc.build_shopping_list(part, 0) {
							sl.Quantity = sl.Quantity * needed
							shopping_list = append(shopping_list, sl)
						}
					}
				}
			}
		}
	}

	tmp := make(map[uint]globalTypes.ShoppingList)
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

	return slices.Collect(maps.Values(tmp))
}

func getRegionCode(region string) (region_coded globalTypes.RegionCode, err error) {
	check_str := strings.ToLower(region)
	switch check_str {
	case "us", "eu", "kr", "tw":
		region_coded = globalTypes.RegionCode(check_str)
	default:
		err = fmt.Errorf("%s is invalid. Valid regions include 'us', 'eu', 'kr', and 'tw'", region)
	}
	return
}

func (cpc *WoWCpCRunner) run(ctx context.Context, region string, server globalTypes.RealmName, useAllProfessions bool, professions_input []globalTypes.CharacterProfession, item globalTypes.ItemSoftIdentity, json_config *globalTypes.RunConfiguration, count uint) (globalTypes.RunReturn, error) {

	cpc.Logger.Info("World of Warcraft Crafting Profit Calculator")

	encoded_region, err := getRegionCode(region)
	if err != nil {
		return globalTypes.RunReturn{Formatted: "NO DATA"}, err
	}

	var professions []globalTypes.CharacterProfession

	if useAllProfessions {
		profList, profErr := cpc.Helper.GetBlizProfessionsList(ctx, encoded_region)
		if profErr != nil {
			return globalTypes.RunReturn{Formatted: "NO DATA"}, profErr
		}
		professions = make([]globalTypes.CharacterProfession, 0, len(profList.Professions))
		for _, prof := range profList.Professions {
			professions = append(professions, globalTypes.CharacterProfession(prof.Name))
		}
	} else {
		professions = professions_input
	}

	price_data, err := cpc.performProfitAnalysis(ctx, encoded_region, server, professions, item, count, nil, nil)
	if err != nil {
		return globalTypes.RunReturn{Formatted: "NO DATA"}, err
	}
	intermediate_data := cpc.generateOutputFormat(ctx, price_data, encoded_region)
	intermediate_data.Shopping_lists = cpc.constructShoppingList(intermediate_data, json_config)
	formatted_data := text_output_helpers.TextFriendlyOutputFormat(&intermediate_data, 0)

	return globalTypes.RunReturn{
		Price:        price_data,
		Intermediate: intermediate_data,
		Formatted:    formatted_data,
	}, nil
}

func (cpc *WoWCpCRunner) RunWithJSONConfig(ctx context.Context, json_config *globalTypes.RunConfiguration) (globalTypes.RunReturn, error) {
	return cpc.run(ctx, json_config.Realm_region, json_config.Realm_name, json_config.UseAllProfessions, json_config.Professions, json_config.Item, json_config, json_config.Item_count)
}

func (cpc *WoWCpCRunner) CliRun(ctx context.Context, json_config *globalTypes.RunConfiguration) error {
	results, err := cpc.RunWithJSONConfig(ctx, json_config)
	if err != nil {
		return err
	}
	return saveOutput(results.Price, results.Intermediate, results.Formatted, cpc.Logger)
}

func saveOutput(price_data globalTypes.ProfitAnalysisObject, intermediate_data globalTypes.OutputFormatObject, formatted_data string, logger *cpclog.CpCLog) error {
	const (
		intermediate_output_fn string = "intermediate_output.json"
		formatted_output_fn    string = "formatted_output"
		raw_output_fn          string = "raw_output.json"
	)

	var errs []error

	logger.Info("Saving output")
	if intermediate_data.Id != 0 {
		if err := func() error {
			intFile, err := os.Create(intermediate_output_fn)
			if err != nil {
				return err
			}
			defer intFile.Close()
			encoder := json.NewEncoder(intFile)
			encoder.SetIndent("", "  ")
			return encoder.Encode(&intermediate_data)
		}(); err != nil {
			errs = append(errs, fmt.Errorf("error saving intermediate output: %w", err))
		}
	}

	if err := func() error {
		forFile, err := os.Create(formatted_output_fn)
		if err != nil {
			return err
		}
		defer forFile.Close()
		formatted_writer := bufio.NewWriter(forFile)
		if _, err := formatted_writer.WriteString(formatted_data); err != nil {
			return err
		}
		return formatted_writer.Flush()
	}(); err != nil {
		errs = append(errs, fmt.Errorf("error saving formatted output: %w", err))
	}

	if price_data.Item_id != 0 {
		if err := func() error {
			rawFile, err := os.Create(raw_output_fn)
			if err != nil {
				return err
			}
			defer rawFile.Close()
			encoder := json.NewEncoder(rawFile)
			encoder.SetIndent("", "  ")
			return encoder.Encode(&price_data)
		}(); err != nil {
			errs = append(errs, fmt.Errorf("error saving raw output: %w", err))
		}
	}

	return errors.Join(errs...)
}

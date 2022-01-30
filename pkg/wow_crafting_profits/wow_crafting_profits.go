package wow_crafting_profits

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/text_output_helpers"
)

type recipeCost struct {
	High, Low uint
	Average   float64
}

/**
 * Find the value of an item on the auction house.
 * Items might be for sale on the auction house and be available from vendors.
 * The auction house items have complicated bonus types.
 * @param {number} item_id The id of the item to search for.
 * @param {object} auction_house An auction house to search through.
 * @param {?number} bonus_level_required An optional bonus level for crafted legendary base items.
 */
func getAHItemPrice(item_id globalTypes.ItemID, auction_house *BlizzardApi.Auctions, bonus_level_required uint) globalTypes.AHItemPriceObject {
	// Find the item and return best, worst, average prices
	auction_high := uint(0)
	auction_low := uint(math.MaxUint)
	auction_average := float64(0)
	auction_counter := uint(0)
	auction_average_accumulator := float64(0)

	for _, auction := range auction_house.Auctions {
		if auction.Item.Id == item_id {

			if ((bonus_level_required != 0) && (len(auction.Item.Bonus_lists) > 0 && auction.Item.Bonus_lists.includes(bonus_level_required))) || (bonus_level_required == 0) {
				if auction.Buyout != 0 {
					if auction.Buyout > auction_high {
						auction_high = auction.Buyout
					}
					if auction.Buyout < auction_low {
						auction_low = auction.Buyout
					}
					auction_average_accumulator += float64(auction.Buyout * auction.Quantity)
				} else {
					if auction.Unit_price > auction_high {
						auction_high = auction.Unit_price
					}
					if auction.Unit_price < auction_low {
						auction_low = auction.Unit_price
					}
					auction_average_accumulator += float64(auction.Unit_price * auction.Quantity)
				}
				auction_counter += auction.Quantity
			}
		}
	}

	auction_average = auction_average_accumulator / float64(auction_counter)

	return globalTypes.AHItemPriceObject{
		High:        auction_high,
		Low:         auction_low,
		Average:     auction_average,
		Total_sales: auction_counter,
	}
}

/**
 * Retrieve the value of the item from the vendor price,
 * items that cannot be bought from
 * vendors are given a value of -1.
 * @param {Number} item_id
 * @param {String} region
 */
func findNoneAHPrice(item_id globalTypes.ItemID, region globalTypes.RegionCode) (float64, error) {
	// Get the item from blizz and see what the purchase price is
	// The general method is to get the item and see if the description mentions the auction house,
	// if it does then return -1, if it doesn't return the 'purchase_price' options
	item, err := blizzard_api_helpers.GetItemDetails(item_id, region)
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

/**
 * Get a list of bonus item values for a given item.
 *
 * Finds all of the bonus-list types associated with a given item id,
 * currently the only way to do that is by pulling an auction house down
 * and then scanning it. If no bonus lists are found an empty array is
 * returned.
 *
 * @param {number} item_id Item ID to scan
 * @param {object} auction_house The auction house data to use as a source.
 */
func getItemBonusLists(item_id globalTypes.ItemID, auction_house BlizzardApi.Auctions) [][]uint {

	array_every := func(array []uint, find []uint) (found bool) {
		found = true
		for index, element := range array {
			found = found || element == find[index]
		}
		return found
	}

	bonus_lists := make([][]uint, 0)
	bonus_lists_set := make([][]uint, 0)
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
			if len(i) == len(list) && array_every(i, list) {
				found = true
			}
		}
		if !found {
			bonus_lists_set = append(bonus_lists_set, list)
		}
	}
	cpclog.Debug("Item ", item_id, " has ", len(bonus_lists_set), " bonus lists.")
	return bonus_lists_set
}

/**
 * Bonus levels correspond to a specific increase in item level over base,
 * get the item level delta for that bonus id.
 * @param bonus_id The bonus ID to check.
 */
func getLvlModifierForBonus(bonus_id uint) uint {
	raidbots_bonus_lists_ptr, _ := static_sources.GetBonuses()
	/*
		if err != nil {
			log.Fatal("unable to get bonus lists")
		}
	*/
	raidbots_bonus_lists := *raidbots_bonus_lists_ptr
	if rbl, present := raidbots_bonus_lists[bonus_id]; present {
		/*if rbl.Level != nil {
			return rbl.Level
		} else {
			return 0
		}*/
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
func performProfitAnalysis(region globalTypes.RegionCode, server globalTypes.RealmName, character_professions []globalTypes.CharacterProfession, item globalTypes.ItemSoftIdentity, qauntity uint, passed_ah *BlizzardApi.Auctions) (globalTypes.ProfitAnalysisObject, error) {
	/*
	   // Check if we have to figure out the item id ourselves
	   let item_id = 0;
	   if (typeof item === 'number') {
	       item_id = item;
	   }
	   else if (Number.isFinite(Number(item))) {
	       item_id = Number(item);
	   } else {
	       item_id = await getItemId(region, item);
	       if (item_id < 0) {
	           logger.error(`No itemId could be found for ${item}`);
	           throw (new Error(`No itemId could be found for ${item}`));
	       }
	       logger.info(`Found ${item_id} for ${item}`);
	   }

	   const item_detail = await getItemDetails(item_id, region);

	   const base_ilvl = item_detail.level;

	   const craftable_item_swaps = await buildCyclicRecipeList(region);

	   let price_obj = {} as ProfitAnalysisObject;
	   price_obj.item_id = item_id;
	   price_obj.item_name = item_detail.name;

	   logger.info(`Analyzing profits potential for ${item_detail.name} (${item_id})`);

	   // Get the realm id
	   const server_id = await getConnectedRealmId(server, region);

	   //Get the auction house
	   const auction_house = (passed_ah !== undefined) ? passed_ah : await getAuctionHouse(server_id, region);

	   // Get Item AH price
	   price_obj.ah_price = await getAHItemPrice(item_id, auction_house);

	   price_obj.item_quantity = qauntity;

	   const item_craftable = await checkIsCrafting(item_id, character_professions, region);

	   // Get NON AH price
	   if (!item_craftable.craftable) {
	       price_obj.vendor_price = await findNoneAHPrice(item_id, region);
	   } else {
	       price_obj.vendor_price = -1;
	   }

	   price_obj.crafting_status = item_craftable;

	   // Eventually bonus_lists should be treated as separate items and this should happen first
	   // When that's the case we should actually return an entire extra set of price data based on each
	   // possible bonus_list. They're actually different items, blizz just tells us they aren't.
	   price_obj.bonus_lists = Array.from(new Set(await getItemBonusLists(item_id, auction_house)));
	   let bonus_link: Record<number, number> = {};
	   const bl_flat = (Array.from(new Set(price_obj.bonus_lists.flat())).filter((bonus: number) => bonus in raidbots_bonus_lists && 'level' in raidbots_bonus_lists[bonus]));
	   for (const bonus of bl_flat) {
	       const mod = getLvlModifierForBonus(bonus);
	       if (mod !== -1) {
	           const new_level = base_ilvl + mod
	           bonus_link[new_level] = bonus;
	           logger.debug(`Bonus level ${bonus} results in crafted ilvl of ${new_level}`);
	       }
	   }

	   const recipe_id_list = item_craftable.recipe_ids.sort();

	   price_obj.recipe_options = [];

	   if (item_craftable.craftable) {
	       logger.debug(`Item ${item_detail.name} (${item_id}) has ${item_craftable.recipes.length} recipes.`);
	       for (const recipe of item_craftable.recipes) {
	           // Get Reagents
	           const item_bom = await getCraftingRecipe(recipe.recipe_id, region);

	           price_obj.item_quantity = qauntity / getRecipeOutputValues(item_bom).min;

	           // Get prices for BOM
	           const bom_prices: ProfitAnalysisObject[] = [];

	           logger.debug(`Recipe ${item_bom.name} (${recipe.recipe_id}) has ${item_bom.reagents.length} reagents`);

	           const bom_promises = item_bom.reagents.map((reagent) => {
	               if (craftable_item_swaps[reagent.reagent.id] !== undefined) {
	                   // We're in a cyclic relationship, what do we do?
	                   logger.error('Cycles are not fully implemented.', craftable_item_swaps[reagent.reagent.id]);
	                   throw new Error( `Cycles are not supported.`);
	               }
	               return performProfitAnalysis(region, server, character_professions, reagent.reagent.id, reagent.quantity, auction_house)
	           });

	           (await Promise.all(bom_promises)).forEach((price) => {
	               bom_prices.push(price);
	           });

	           let rank_level = 0;
	           let rank_AH = {} as AHItemPriceObject;
	           if (recipe_id_list.length > 1) {
	               rank_level = recipe_id_list.indexOf(recipe.recipe_id) > -1 ? rankings.available_levels[rankings.rank_mapping[recipe_id_list.indexOf(recipe.recipe_id)]] : 0;
	               if (bonus_link[rank_level] != undefined) {
	                   logger.debug(`Looking for AH price for ${item_id} for level ${rank_level} using bonus is ${bonus_link[rank_level]}`);
	                   rank_AH = await getAHItemPrice(item_id, auction_house, bonus_link[rank_level]);
	               } else {
	                   logger.debug(`Item ${item_id} has no auctions for level ${rank_level}`);
	               }
	           }

	           price_obj.recipe_options.push({
	               recipe: recipe,
	               prices: bom_prices,
	               rank: rank_level,
	               rank_ah: rank_AH,
	           });
	       }
	   } else {
	       logger.debug(`Item ${item_detail.name} (${item_id}) not craftable with professions: ${character_professions}`);
	       if (price_obj.bonus_lists.length > 0) {
	           price_obj.bonus_prices = [];
	           for (const bonus of bl_flat) {
	               const rbl = raidbots_bonus_lists[bonus];
	               const level_uncrafted_ah_cost = {
	                   level: base_ilvl + (rbl.level !== undefined ? rbl.level : 0),
	                   ah: await getAHItemPrice(item_id, auction_house, bonus)
	               };
	               price_obj.bonus_prices.push(level_uncrafted_ah_cost);
	           }
	       }
	   }

	   return price_obj;
	*/
	return globalTypes.ProfitAnalysisObject{}, fmt.Errorf("not implemented")
}

/**
 * Figure out the best/worst/average cost to construct a recipe given all items required.
 * @param recipe_option The recipe to price.
 */
func recipeCostCalculator(recipe_option globalTypes.RecipeOption) recipeCost {
	/**
	 * For each recipe
	 *   For each component
	 *     if component is vendor: cost = price * quantity
	 *     if component is on AH: cost = h/l/a * quantity (tuple)
	 *     if component is craftable: cost = h/l/a of each recipe option
	 */
	cost := recipeCost{}

	for _, component := range recipe_option.Prices {
		if component.Vendor_price != 0 {
			cost.High += component.Vendor_price * component.Item_quantity
			cost.Low += component.Vendor_price * component.Item_quantity
			cost.Average += float64(component.Vendor_price * component.Item_quantity)
			cpclog.Debug("Use vendor price for ", component.Item_name, " (", component.Item_id, ")")
		} else if component.Crafting_status.Craftable == false {

			high := uint(0)
			low := uint(math.MaxUint64)
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
			cpclog.Debugf("Use auction price for uncraftable item %s (%s)", component.Item_name, component.Item_id)
		} else {
			cpclog.Debugf("Recursive check for item %s (%s)", component.Item_name, component.Item_id)
			ave_acc := float64(0)
			ave_cnt := 0

			high := uint(0)
			low := uint(math.MaxUint64)

			for _, opt := range component.Recipe_options {
				//rc_price_promises.push(recipeCostCalculator(opt));
				recurse_price := recipeCostCalculator(opt)

				if high < recurse_price.High*component.Item_quantity {
					high = recurse_price.High * component.Item_quantity
				}

				if low > recurse_price.Low*component.Item_quantity {
					low = recurse_price.Low * component.Item_quantity
				}

				ave_acc += recurse_price.Average * float64(component.Item_quantity)
				ave_cnt++
			}

			cost.Low = low
			cost.High = high
			cost.Average += ave_acc / float64(ave_cnt)
		}
	}

	return cost
}

/**
 * Create an object used for constructing shopping lists and formatted output data.
 * @param {!object} price_data The object created by the analyze function.
 * @param {!string} region The region in which to work.
 */
func generateOutputFormat(price_data globalTypes.ProfitAnalysisObject, region globalTypes.RegionCode) globalTypes.OutputFormatObject {
	/*
				const object_output = {} as OutputFormatObject;
		        object_output.name = price_data.item_name;
		        object_output.id = price_data.item_id;
		        object_output.required = price_data.item_quantity;
		        object_output.recipes = [];

		        if ((price_data.ah_price != undefined) && (price_data.ah_price.total_sales > 0)) {
		            object_output.ah = {
		                sales: price_data.ah_price.total_sales,
		                high: price_data.ah_price.high,
		                low: price_data.ah_price.low,
		                average: price_data.ah_price.average,
		            }
		        }
		        if (price_data.vendor_price > 0) {
		            object_output.vendor = price_data.vendor_price;
		        }
		        if (price_data.recipe_options != undefined) {
		            for (let recipe_option of price_data.recipe_options) {
		                const option_price = await recipeCostCalculator(recipe_option);
		                const recipe = await getBlizRecipeDetail(recipe_option.recipe.recipe_id, region);
		                const obj_recipe = {} as OutputFormatObject["recipes"][number];
		                obj_recipe.name = recipe.name;
		                obj_recipe.rank = recipe_option.rank;
		                obj_recipe.id = recipe_option.recipe.recipe_id;
		                obj_recipe.output = getRecipeOutputValues(recipe);
		                obj_recipe.high = option_price.high;
		                obj_recipe.low = option_price.low;
		                obj_recipe.average = option_price.average;
		                obj_recipe.parts = [];

		                if ((recipe_option.rank_ah != undefined) && (recipe_option.rank_ah.total_sales > 0)) {
		                    obj_recipe.ah = {
		                        sales: recipe_option.rank_ah.total_sales,
		                        high: recipe_option.rank_ah.high,
		                        low: recipe_option.rank_ah.low,
		                        average: recipe_option.rank_ah.average,
		                    };
		                }
		                let prom_list = [];
		                if (recipe_option.prices != undefined) {
		                    for (let opt of recipe_option.prices) {
		                        prom_list.push(generateOutputFormat(opt, region));
		                    }
		                    (await Promise.all(prom_list)).forEach((data) => {
		                        obj_recipe.parts.push(data);
		                    });
		                }

		                object_output.recipes.push(obj_recipe);
		            }
		        }

		        if (price_data.bonus_prices !== undefined) {
		            object_output.bonus_prices = price_data.bonus_prices.map((bonus_price) => {
		                return {
		                    level: bonus_price.level,
		                    ah: {
		                        sales: bonus_price.ah.total_sales,
		                        high: bonus_price.ah.high,
		                        low: bonus_price.ah.low,
		                        average: bonus_price.ah.average,
		                    }
		                };
		            })
		        }

		        return object_output;
	*/
}

/*
       "crafted_quantity": {
           "minimum": 1,
           "maximum": 1
       }

   OR

       "crafted_quantity": {
           "value": 3
       }
*/

func getRecipeOutputValues(recipe BlizzardApi.Recipe) globalTypes.OutpoutFormatRecipeOutput {
	min, max, value := -1, -1, -1
	if recipe.Crafted_quantity.Minimum != 0 {
		min = int(recipe.Crafted_quantity.Minimum)
	}
	if recipe.Crafted_quantity.Maximum != 0 {
		max = int(recipe.Crafted_quantity.Maximum)
	}
	if recipe.Crafted_quantity.Value != 0 {
		value = int(recipe.Crafted_quantity.Value)
	}

	if min == -1 && max == -1 {
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
func constructShoppingList(intermediate_data globalTypes.OutputFormatObject, on_hand globalTypes.RunConfiguration) globalTypes.OutputFormatShoppingList {
	var shopping_lists globalTypes.OutputFormatShoppingList
	for _, rank := range getShoppingListRanks(intermediate_data) {
		cpclog.Debug("Resetting inventory for rank shopping list.")
		on_hand.ResetInventoryAdjustments()
		shopping_list := build_shopping_list(intermediate_data, rank)
		for _, li := range shopping_list {
			needed := li.Quantity
			available := on_hand.ItemCount(li.Id)

			cpclog.Debugf("%s (%s) %s needed with %s available", li.Name, li.Id, needed, available)
			if needed <= available {
				cpclog.Debugf("$%s (%s) used %s of the available %s", li.Name, li.Id, needed, available)
				needed = 0
				on_hand.AdjustInventory(li.Id, (int(needed) * -1))
			} else if (needed > available) && (int(available) != 0) {
				needed -= available
				cpclog.Debugf("%s (%s) used all of the available %s and still need %s", li.Name, li.Id, available, needed)
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
				li.Cost.Ah.Average *= float64(li.Quantity)
			}
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
func build_shopping_list(intermediate_data globalTypes.OutputFormatObject, rank_requested uint) []globalTypes.ShoppingList {
	/*
	   let shopping_list = [];

	   logger.debug(`Build shopping list for ${intermediate_data.name} (${intermediate_data.id}) rank ${rank_requested}`);

	   let needed = intermediate_data.required;

	   if (intermediate_data.recipes.length === 0) {
	       shopping_list.push({
	           id: intermediate_data.id,
	           name: intermediate_data.name,
	           quantity: intermediate_data.required,
	           cost: {
	               ah: intermediate_data.ah,
	               vendor: intermediate_data.vendor,
	           },
	       });
	       logger.debug(`${intermediate_data.name} (${intermediate_data.id}) cannot be crafted.`);
	   } else {
	       for (let recipe of intermediate_data.recipes) {
	           // Make sure the recipe isn't on the exclusion list
	           if (shopping_recipe_exclusions.exclusions.includes(recipe.id)) {
	               logger.debug(`${recipe.name} (${recipe.id}) is on the exclusion list. Add it directly`);
	               shopping_list.push({
	                   id: intermediate_data.id,
	                   name: intermediate_data.name,
	                   quantity: intermediate_data.required,
	                   cost: {
	                       ah: intermediate_data.ah,
	                       vendor: intermediate_data.vendor,
	                   },
	               });
	           } else {
	               if (recipe.rank == rank_requested) {
	                   for (let part of recipe.parts) {
	                       // Only top level searches can have ranks
	                       build_shopping_list(part, 0).forEach((sl) => {
	                           //let al = sl;
	                           logger.debug(`Need ${sl.quantity} of ${sl.name} (${sl.id}) for each of ${needed}`);

	                           sl.quantity = sl.quantity * needed;

	                           shopping_list.push(sl);
	                       });
	                   }
	               } else {
	                   logger.debug(`Skipping recipe ${recipe.id} because its rank (${recipe.rank}) does not match the requested rank (${rank_requested})`);
	               }
	           }
	       }
	   }

	   // Build the return shopping list.
	   let tmp: Record<number | string, ShoppingList> = {};
	   let ret_list: ShoppingList[] = [];
	   //logger.debug(shopping_list);
	   for (let list_element of shopping_list) {
	       if (!(list_element.id in tmp)) {
	           tmp[list_element.id] = {
	               id: list_element.id,
	               name: list_element.name,
	               quantity: 0,
	               cost: list_element.cost,
	           };
	       }
	       tmp[list_element.id].quantity += list_element.quantity;
	   }
	   Object.keys(tmp).forEach((id) => {
	       ret_list.push(tmp[id]);
	   });

	   return ret_list;
	*/
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
		err = fmt.Errorf("%s is invalid. Valid regions include 'us', 'eu', 'kr', and 'tw'.", region)
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
func run(region string, server globalTypes.RealmName, professions []globalTypes.CharacterProfession, item globalTypes.ItemSoftIdentity, json_config globalTypes.RunConfiguration, count uint) (globalTypes.RunReturn, error) {

	cpclog.Info("World of Warcraft Crafting Profit Calculator")

	cpclog.Infof("Checking %s in %s for %s with available professions %s", server, region, item, professions)

	//let formatted_data = 'NO DATA';

	encoded_region, err := getRegionCode(region)
	if err != nil {
		return globalTypes.RunReturn{Formatted: "NO DATA"}, err
	}

	price_data, err := performProfitAnalysis(encoded_region, server, professions, item, count, nil)
	if err != nil {
		return globalTypes.RunReturn{Formatted: "NO DATA"}, err
	}
	intermediate_data := generateOutputFormat(price_data, encoded_region)
	intermediate_data.Shopping_lists = constructShoppingList(intermediate_data, json_config)
	formatted_data := text_output_helpers.TextFriendlyOutputFormat(intermediate_data, 0)

	return globalTypes.RunReturn{
		Price:        price_data,
		Intermediate: intermediate_data,
		Formatted:    formatted_data,
	}, nil

	return globalTypes.RunReturn{}, fmt.Errorf("not implemented")
}

/**
 * Save the generated output to the filesystem.
 * @param price_data The price data.
 * @param intermediate_data The output cost object with shopping list.
 * @param formatted_data The preformatted text output with shopping list.
 */
func saveOutput(price_data globalTypes.ProfitAnalysisObject, intermediate_data globalTypes.OutputFormatObject, formatted_data string) error {
	const (
		intermediate_output_fn string = "intermediate_output.json"
		formatted_output_fn    string = "formatted_output"
		raw_output_fn          string = "raw_output.json"
	)
	cpclog.Info("Saving output")
	if intermediate_data.Id != 0 {
		intFile, err := os.Create(intermediate_output_fn)
		if err != nil {
			return err
		}
		defer intFile.Close()
		encode_err := json.NewEncoder(intFile).Encode(&intermediate_data)
		if encode_err != nil {
			return encode_err
		}
		cpclog.Info("Intermediate output saved")
	}
	forFile, err := os.Create(raw_output_fn)
	if err != nil {
		return err
	}
	defer forFile.Close()
	forFile.WriteString(formatted_data)
	cpclog.Info("Formatted output saved")
	if price_data.Item_id != 0 {
		rawFile, err := os.Create(raw_output_fn)
		if err != nil {
			return err
		}
		defer rawFile.Close()
		encode_err := json.NewEncoder(rawFile).Encode(&price_data)
		if encode_err != nil {
			return encode_err
		}
		cpclog.Info("Raw output saved")
	}
	return fmt.Errorf("not implemented")
}

/**
 * Perform a run with pure json configuration from the addon.
 * @param {RunConfiguration} json_config The configuration object.
 */
func RunWithJSONConfig(json_config globalTypes.RunConfiguration) (globalTypes.RunReturn, error) {
	return run(json_config.Realm_region, json_config.Realm_name, json_config.Professions, json_config.Item, json_config, json_config.Item_count)
	//return globalTypes.RunReturn{}, fmt.Errorf("not implemented")
}

/**
 * Run from the command prompt.
 * @param {RunConfiguration} json_config The configuration object to execute.
 */
func CliRun(json_config globalTypes.RunConfiguration) error {
	results, err := RunWithJSONConfig(json_config)
	if err != nil {
		return err
	}
	saveOutput(results.Price, results.Intermediate, results.Formatted)
	return nil
}

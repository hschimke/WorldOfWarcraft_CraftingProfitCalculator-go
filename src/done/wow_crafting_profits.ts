import { promises as fs } from 'fs';
import { Logger } from 'winston';
import { CPCApiHelpers } from './blizzard-api-helpers.js';
import { static_sources } from './cached-data-sources.js';
import { getRegionCode } from './getRegionCode.js';
import { RunConfiguration } from './RunConfiguration.js';
import { textFriendlyOutputFormat } from './text-output-helpers.js';

async function CPCInstance(logging: Logger, cache: CPCCache, api: CPCApi) {
    const logger = logging;

    const cached_static_resources = await static_sources();
    const raidbots_bonus_lists: BonusesCache = cached_static_resources.bonuses_cache;
    const rankings: RankMappingsCache = cached_static_resources.rank_mappings_cache;
    const shopping_recipe_exclusions: ShoppingRecipeExclusionList = cached_static_resources.shopping_recipe_exclusion_list;

    const { getItemId, getConnectedRealmId, getItemDetails, getBlizRecipeDetail, checkIsCrafting, getCraftingRecipe, getAuctionHouse, buildCyclicRecipeList } = CPCApiHelpers(logger, cache, api);

    /**
     * Find the value of an item on the auction house.
     * Items might be for sale on the auction house and be available from vendors.
     * The auction house items have complicated bonus types.
     * @param {number} item_id The id of the item to search for.
     * @param {object} auction_house An auction house to search through.
     * @param {?number} bonus_level_required An optional bonus level for crafted legendary base items.
     */
    async function getAHItemPrice(item_id: ItemID, auction_house: BlizzardApi.Auctions, bonus_level_required?: number): Promise<AHItemPriceObject> {
        // Find the item and return best, worst, average prices

        let auction_high = Number.MIN_VALUE;
        let auction_low = Number.MAX_VALUE;
        let auction_average = 0;
        let auction_counter = 0;
        let auction_average_accumulator = 0;

        /**
         * Issue: for items with multiple version availble on the auction house
         * you will find that they all have the same id. The difference comes
         * from the 'bonus_lists' array
         */
        auction_house.auctions.forEach((auction) => {
            if (auction.item.id == item_id) {
                if (((bonus_level_required != undefined) && (('bonus_lists' in auction.item) && auction.item.bonus_lists.includes(bonus_level_required))) || (bonus_level_required == undefined)) {
                    if (auction.buyout !== undefined) {
                        if (auction.buyout > auction_high) {
                            auction_high = auction.buyout;
                        }
                        if (auction.buyout < auction_low) {
                            auction_low = auction.buyout;
                        }
                        auction_average_accumulator += (auction.buyout * auction.quantity);
                    } else {
                        if (auction.unit_price > auction_high) {
                            auction_high = auction.unit_price;
                        }
                        if (auction.unit_price < auction_low) {
                            auction_low = auction.unit_price;
                        }
                        auction_average_accumulator += (auction.unit_price * auction.quantity);
                    }
                    auction_counter += auction.quantity;
                }
            }
        });

        auction_average = auction_average_accumulator / auction_counter;

        return {
            high: auction_high,
            low: auction_low,
            average: auction_average,
            total_sales: auction_counter,
        };
    }

    /**
     * Retrieve the value of the item from the vendor price,
     * items that cannot be bought from
     * vendors are given a value of -1.
     * @param {Number} item_id 
     * @param {String} region 
     */
    async function findNoneAHPrice(item_id: ItemID, region: RegionCode): Promise<number> {
        // Get the item from blizz and see what the purchase price is
        // The general method is to get the item and see if the description mentions the auction house,
        // if it does then return -1, if it doesn't return the 'purchase_price' options
        const item = await getItemDetails(item_id, region);
        let vendor_price = -1;
        if (item.description !== undefined) {
            if (item.description.includes('vendor')) {
                vendor_price = item.purchase_price;
            }
            if (!item.description.includes('auction')) {
                vendor_price = item.purchase_price;
            }
        } else {
            vendor_price = item.purchase_price;
        }
        if (item.purchase_quantity !== undefined) {
            vendor_price = vendor_price / item.purchase_quantity;
        }
        return vendor_price;
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
    async function getItemBonusLists(item_id: ItemID, auction_house: BlizzardApi.Auctions): Promise<Array<Array<number>>> {
        let bonus_lists: Array<Array<number>> = [];
        let bonus_lists_set: Array<Array<number>> = [];
        auction_house.auctions.forEach((auction) => {
            if (auction.item.id == item_id) {
                if ('bonus_lists' in auction.item) {
                    bonus_lists.push(auction.item.bonus_lists);
                }
            }
        });
        bonus_lists.forEach((list) => {
            let found = false;
            bonus_lists_set.forEach((i) => {
                if (i.length == list.length && i.every(function (u, i) {
                    return u === list[i];
                })
                ) {
                    found = true;
                }
            });
            if (!found) {
                bonus_lists_set.push(list);
            }
        });
        logger.debug(`Item ${item_id} has ${bonus_lists_set.length} bonus lists.`);
        return bonus_lists_set;
    }

    /**
     * Bonus levels correspond to a specific increase in item level over base,
     * get the item level delta for that bonus id.
     * @param bonus_id The bonus ID to check.
     */
    function getLvlModifierForBonus(bonus_id: number): number {
        if (bonus_id in raidbots_bonus_lists) {
            const rbl = raidbots_bonus_lists[bonus_id];
            if (rbl.level !== undefined) {
                return rbl.level;
            } else {
                return -1;
            }
        } else {
            return -1;
        }
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
    async function performProfitAnalysis(region: RegionCode, server: RealmName, character_professions: Array<CharacterProfession>, item: ItemSoftIdentity, qauntity: number, passed_ah?: BlizzardApi.Auctions): Promise<ProfitAnalysisObject> {
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
    }

    /**
     * Figure out the best/worst/average cost to construct a recipe given all items required.
     * @param recipe_option The recipe to price.
     */
    async function recipeCostCalculator(recipe_option: ProfitAnalysisObject["recipe_options"][number]): Promise<{ high: number, low: number, average: number }> {
        /**
         * For each recipe
         *   For each component
         *     if component is vendor: cost = price * quantity
         *     if component is on AH: cost = h/l/a * quantity (tuple)
         *     if component is craftable: cost = h/l/a of each recipe option
         */
        const cost = {
            high: 0,
            low: 0,
            average: 0
        };

        for (let component of recipe_option.prices) {
            if (component.vendor_price != -1) {
                cost.high += component.vendor_price * component.item_quantity;
                cost.low += component.vendor_price * component.item_quantity;
                cost.average += component.vendor_price * component.item_quantity;
                logger.debug(`Use vendor price for ${component.item_name} (${component.item_id})`);
            } else if (component.crafting_status.craftable == false) {
                let high = Number.MIN_VALUE;
                let low = Number.MAX_VALUE;
                let average = 0;
                let count = 0;
                if (component.ah_price.total_sales > 0) {
                    average += component.ah_price.average;
                    if (component.ah_price.high > high) {
                        high = component.ah_price.high;
                    }
                    if (component.ah_price.low < low) {
                        low = component.ah_price.low;
                    }
                    count++;
                }
                cost.average += (average / count) * component.item_quantity;
                cost.high += high * component.item_quantity;
                cost.low += low * component.item_quantity;
                logger.debug(`Use auction price for uncraftable item ${component.item_name} (${component.item_id})`);
            } else {
                logger.debug(`Recursive check for item ${component.item_name} (${component.item_id})`);
                let ave_acc = 0;
                let ave_cnt = 0;

                let high = Number.MIN_VALUE;
                let low = Number.MAX_VALUE;

                let rc_price_promises = [];
                for (const opt of component.recipe_options) {
                    rc_price_promises.push(recipeCostCalculator(opt));
                }

                (await Promise.all(rc_price_promises)).forEach((recurse_price) => {
                    if (high < recurse_price.high * component.item_quantity) {
                        high = recurse_price.high * component.item_quantity;
                    }

                    if (low > recurse_price.low * component.item_quantity) {
                        low = recurse_price.low * component.item_quantity;
                    }

                    ave_acc += recurse_price.average * component.item_quantity;
                    ave_cnt++;
                });

                cost.low = low;
                cost.high = high;
                cost.average += ave_acc / ave_cnt;
            }
        }

        return cost;
    }

    /**
     * Create an object used for constructing shopping lists and formatted output data.
     * @param {!object} price_data The object created by the analyze function.
     * @param {!string} region The region in which to work.
     */
    async function generateOutputFormat(price_data: ProfitAnalysisObject, region: RegionCode): Promise<OutputFormatObject> {
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

    function getRecipeOutputValues(recipe: BlizzardApi.Recipe): OutputFormatObject["recipes"][number]["output"] {
        let min = -1;
        let max = -1;
        let value = -1;
        if (recipe.crafted_quantity.minimum !== undefined) {
            min = recipe.crafted_quantity.minimum;
        }
        if (recipe.crafted_quantity.maximum !== undefined) {
            max = recipe.crafted_quantity.maximum;
        }
        if (recipe.crafted_quantity.value !== undefined) {
            value = recipe.crafted_quantity.value;
        }

        if (min === -1 && max === -1) {
            min = value;
            max = value;
        }

        return { min, max, value };
    }

    /**
     * Return the ranks available for the top level item generated from generateOutputFormat.
     * @param {!object} intermediate_data Data from generateOutputFormat.
     */
    function getShoppingListRanks(intermediate_data: OutputFormatObject): Array<number> {
        const ranks = [];
        for (let recipe of intermediate_data.recipes) {
            ranks.push(recipe.rank);
        }
        return ranks;
    }

    /**
     * Construct a shopping list given a provided inventory object.
     * @param {!object} intermediate_data Data from generateOutputFormat.
     * @param {!RunConfiguration} on_hand A provided inventory to get existing items from.
     */
    function constructShoppingList(intermediate_data: OutputFormatObject, on_hand: RunConfiguration): OutputFormatShoppingList {
        const shopping_lists: OutputFormatShoppingList = {} as OutputFormatShoppingList;
        for (const rank of getShoppingListRanks(intermediate_data)) {
            logger.debug(`Resetting inventory for rank shopping list.`);
            on_hand.resetInventoryAdjustments();
            const shopping_list = build_shopping_list(intermediate_data, rank);
            for (const li of shopping_list) {
                let needed = li.quantity;
                let available = on_hand.itemCount(li.id);

                logger.debug(`${li.name} (${li.id}) ${needed} needed with ${available} available`);
                if (needed <= available) {
                    logger.debug(`${li.name} (${li.id}) used ${needed} of the available ${available}`);
                    needed = 0;
                    on_hand.adjustInventory(li.id, (needed * -1));
                } else if ((needed > available) && (available != 0)) {
                    needed -= available;
                    logger.debug(`${li.name} (${li.id}) used all of the available ${available} and still need ${needed}`);
                    on_hand.adjustInventory(li.id, (available * -1));
                }

                li.quantity = needed;

                // Update the cost for this list item
                if (li.cost.vendor !== undefined) {
                    li.cost.vendor *= li.quantity;
                }
                if (li.cost.ah !== undefined) {
                    li.cost.ah.high *= li.quantity;
                    li.cost.ah.low *= li.quantity;
                    li.cost.ah.average *= li.quantity;
                }
            }
            shopping_lists[rank] = shopping_list;
        }
        return shopping_lists;
    }

    /**
     * Build a raw shopping list using generateOutputFormat data, ignores inventory information.
     * @param {!object} intermediate_data The generateOutputFormat data used for construction.
     * @param {number} rank_requested The specific rank to generate a list for, only matters for legendary base items in Shadowlands.
     */
    function build_shopping_list(intermediate_data: OutputFormatObject, rank_requested: number): Array<ShoppingList> {
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
    async function run(region: string, server: RealmName, professions: Array<CharacterProfession>, item: ItemSoftIdentity, json_config: RunConfiguration, count: number): Promise<RunReturn> {
        logger.info("World of Warcraft Crafting Profit Calculator");

        logger.info(`Checking ${server} in ${region} for ${item} with available professions ${JSON.stringify(professions)}`);

        let intermediate_data: OutputFormatObject | undefined = undefined;
        let price_data: ProfitAnalysisObject | undefined = undefined
        let formatted_data = 'NO DATA';

        try {
            price_data = await performProfitAnalysis(getRegionCode(region), server, professions, item, count);
            intermediate_data = await generateOutputFormat(price_data, getRegionCode(region));
            intermediate_data.shopping_lists = constructShoppingList(intermediate_data, json_config);
            formatted_data = await textFriendlyOutputFormat(intermediate_data, 0);
        } catch (e) {
            logger.error(`Error building output for ${region}:${server} ${item}`, e);
            throw(e);
        }

        return {
            price: price_data,
            intermediate: intermediate_data,
            formatted: formatted_data,
        }
    }

    /**
     * Save the generated output to the filesystem.
     * @param price_data The price data.
     * @param intermediate_data The output cost object with shopping list.
     * @param formatted_data The preformatted text output with shopping list.
     */
    async function saveOutput(price_data: ProfitAnalysisObject | undefined, intermediate_data: OutputFormatObject | undefined, formatted_data: string): Promise<void> {
        logger.info('Saving output');
        if (intermediate_data !== undefined) {
            await fs.writeFile('intermediate_output.json', JSON.stringify(intermediate_data, null, 2), 'utf8');
            logger.info('Intermediate output saved');
        }
        await fs.writeFile('formatted_output', formatted_data, 'utf8');
        logger.info('Formatted output saved');
        if (price_data !== undefined) {
            await fs.writeFile('raw_output.json', JSON.stringify(price_data, null, 2), 'utf8');
            logger.info('Raw output saved');
        }
    }

    /**
     * Perform a run with pure json configuration from the addon.
     * @param {RunConfiguration} json_config The configuration object.
     */
    async function runWithJSONConfig(json_config: RunConfiguration): Promise<RunReturn> {
        return run(json_config.realm_region,
            json_config.realm_name,
            json_config.professions,
            json_config.item_id,
            json_config,
            json_config.item_count
        );
    }

    /**
     * Run from the command prompt.
     * @param {RunConfiguration} json_config The configuration object to execute.
     */
    async function cliRun(json_config: RunConfiguration): Promise<void> {
        try {
            const { price, intermediate, formatted } = await runWithJSONConfig(json_config);
            await saveOutput(price, intermediate, formatted);
        } finally {
        }
    }

    return Object.freeze({
        runWithJSONConfig,
        cliRun
    });
}

export { CPCInstance };

import { Logger } from 'winston';
import { getComputedTimeWithShift, getDynamicTimeWithShift, getStaticTimeWithShift } from './cache_time_helpers.js';

const exclude_before_shadowlands = false;

// Cache namespace constants
const ITEM_SEARCH_CACHE = 'item_search_cache';
const CONNECTED_REALM_ID_CACHE = 'connected_realm_data';
const ITEM_DATA_CACHE = 'fetched_item_data';
const PROFESSION_SKILL_TIER_DETAILS_CACHE = 'fetched_profession_skill_tier_detail_data';
const PROFESSION_RECIPE_DETAIL_CACHE = 'fetched_profession_recipe_detail_data';
const CRAFTABLE_BY_PROFESSION_SET_CACHE = 'craftable_by_professions_cache';
const CRAFTABLE_BY_SINGLE_PROFESSION_CACHE = 'craftable_by_profession';
const AUCTION_DATA_CACHE = 'fetched_auctions_data';
const PROFESSION_DETAIL_CACHE = 'profession_detail_data';
const PROFESSION_LIST_CACHE = 'regional_profession_list';
const COMPOSITE_REALM_NAME_CACHE = 'connected_realm_detail';
const CYCLIC_LINK_CACHE = 'cyclic_links';

function CPCApiHelpers(logging: Logger, cache: CPCCache, api: CPCApi): CPCApiHelpers {
    const logger = logging;
    const { cacheCheck, cacheGet, cacheSet } = cache;
    const { getBlizzardAPIResponse, getBlizzardRawUriResponse } = api;
    /**
     * Search through the item database for a string, returning the item id of the item.
     * @param {!string} region The region in which to search.
     * @param {!string} item_name The name of the item to search for
     */
    async function getItemId(region: RegionCode, item_name: ItemName): Promise<ItemID> {
        logger.debug(`Searching for itemId for ${item_name}`);

        if (await cacheCheck(ITEM_SEARCH_CACHE, item_name)) {
            return cacheGet(ITEM_SEARCH_CACHE, item_name);
        }

        const search_api_uri = '/data/wow/search/item';
        let item_id = -1;

        let page_counter = 0;
        for await (const pg of getAllSearchPages()) {
            page_counter++;
            let page_item_id = await checkPageSearchResults(pg, item_name);
            if (page_item_id > 0) {
                item_id = page_item_id;
                logger.debug(`Found ${item_id} for ${item_name} on page ${page_counter} of ${pg.pageCount}.`);
                break;
            }
        }

        //if (item_id > 0) {
        await cacheSet(ITEM_SEARCH_CACHE, item_name, item_id, getStaticTimeWithShift());
        //}

        return item_id;

        /**
         * Search a specific page for the item.
         * @param {!number} page The page to retrieve and search.
         * @param {!string} item_name The name of the item we are searching for.
         * @return The item id of the item, or -1 if it doesn't.
         */
        async function checkPageSearchResults(page: BlizzardApi.ItemSearch, item_name: ItemName): Promise<ItemID> {
            let found_item_id = -1;
            for (let result of page.results) {
                //console.log(result);
                if (result.data.name['en_US'].localeCompare(item_name, undefined, { sensitivity: 'accent' }) == 0) {
                    logger.debug(`Found ${item_name} with id ${result.data.id}`);
                    found_item_id = result.data.id;
                    break;
                }
            }
            return found_item_id;
        }

        /**
         * Create an interator for all pages of the search.
         */
        async function* getAllSearchPages() {
            // Step 1: Get the initial results to see if we get anything
            const initial_page: BlizzardApi.ItemSearch = <BlizzardApi.ItemSearch>await getBlizzardAPIResponse(region, {
                'namespace': getNamespace('static', region),
                'locale': 'en_US',
                'name.en_US': item_name,
                'orderby': 'id:desc',
            },
                search_api_uri);
            const page_count = initial_page.pageCount;
            logger.debug(`Found ${page_count} pages for item search ${item_name}`);
            if (page_count > 0) {
                yield initial_page;
                for (let cp = initial_page.page; cp <= page_count; cp++) {
                    logger.debug(`Checking page ${cp} for ${item_name}`);
                    const current_page = <BlizzardApi.ItemSearch>await getBlizzardAPIResponse(region, {
                        'namespace': getNamespace('static', region),
                        'locale': 'en_US',
                        'name.en_US': item_name,
                        'orderby': 'id:desc',
                        '_page': cp,
                    },
                        search_api_uri);
                    yield current_page;
                }
            } else {
                // We didn't get any results, that's an error
                //await cacheSet(ITEM_SEARCH_CACHE, item_name, -1);
                logger.error(`No items match search ${item_name}`);
                throw (new Error('No Results'));
            }
        }
    }

    /**
     * Return a list of all connected realms available within a region.
     * @param {string} region Region to return list of connected realms.
     * @param {object} token An OATH access token to use for the request.
     */
    async function getAllConnectedRealms(region: RegionCode): Promise<BlizzardApi.ConnectedRealmIndex> {
        const list_connected_realms_api = '/data/wow/connected-realm/index';
        const list_connected_realms_form = {
            'namespace': getNamespace('dynamic', region),
            'locale': 'en_US'
        };

        return <Promise<BlizzardApi.ConnectedRealmIndex>>getBlizzardAPIResponse(
            region,
            list_connected_realms_form,
            list_connected_realms_api);
    }

    /**
     * Find the connected realm id for a server, some connected realms have only one server
     * while others have multiples.
     * @param {!string} server_name The name of the server to look for.
     * @param {!string} server_region The region in which the server is hosted.
     * @return {Promise.<number>} The number associated with the connected realm for the server.
     */
    async function getConnectedRealmId(server_name: RealmName, server_region: RegionCode): Promise<ConnectedRealmID> {
        const connected_realm_key = `${server_region}::${server_name}`;

        if (await cacheCheck(CONNECTED_REALM_ID_CACHE, connected_realm_key)) {
            return cacheGet(CONNECTED_REALM_ID_CACHE, connected_realm_key);
        }

        const get_connected_realm_form = {
            'namespace': getNamespace('dynamic', server_region),
            'locale': 'en_US'
        };

        let realm_id = 0;

        // Get a list of all connected realms
        const all_connected_realms = await getAllConnectedRealms(server_region);

        // Pull the data for each connection until you find one with the server name in question
        for (let realm_href of all_connected_realms.connected_realms) {
            const hr = realm_href.href;
            const connected_realm_detail = <BlizzardApi.ConnectedRealm>await getBlizzardRawUriResponse(get_connected_realm_form, hr, server_region);
            const realm_list = connected_realm_detail.realms;
            let found_realm = false;
            for (let rlm of realm_list) {
                logger.debug(`Realm ${rlm.name}`);
                if (rlm.name.localeCompare(server_name, undefined, { sensitivity: 'accent' }) == 0) {
                    logger.debug(`Realm ${rlm} matches ${server_name}`);
                    found_realm = true;
                    break;
                }
            }
            if (found_realm == true) {
                realm_id = connected_realm_detail.id;
                break;
            }
        }

        await cacheSet(CONNECTED_REALM_ID_CACHE, connected_realm_key, realm_id, getStaticTimeWithShift());
        logger.info(`Found Connected Realm ID: ${realm_id} for ${server_region} ${server_name}`);

        // Return that connected realm ID
        return realm_id;
    }

    /**
     * Get the full details for an item.
     * @param {!number} item_id The item id for the item to retrieve.
     * @param {!string} region The region in which we are searching.
     */
    async function getItemDetails(item_id: ItemID, region: RegionCode): Promise<BlizzardApi.Item> {
        const key = item_id

        if (await cacheCheck(ITEM_DATA_CACHE, key)) {
            return cacheGet(ITEM_DATA_CACHE, key);
        }

        const profession_item_detail_uri = `/data/wow/item/${item_id}`;
        //categories[array].recipes[array].name categories[array].recipes[array].id
        const result = <BlizzardApi.Item>await getBlizzardAPIResponse(region, {
            'namespace': getNamespace('static', region),
            'locale': 'en_US'
        },
            profession_item_detail_uri);

        await cacheSet(ITEM_DATA_CACHE, key, result, getStaticTimeWithShift());
        return result;
    }

    /**
     * Get a list of all professions available.
     * @param {!string} region The region to search
     */
    async function getBlizProfessionsList(region: RegionCode): Promise<BlizzardApi.ProfessionsIndex> {
        const key = region;
        const profession_list_uri = '/data/wow/profession/index'; // professions.name / professions.id

        if (await cacheCheck(PROFESSION_LIST_CACHE, key)) {
            return cacheGet(PROFESSION_LIST_CACHE, key);
        }

        const result = <BlizzardApi.ProfessionsIndex>await getBlizzardAPIResponse(region, {
            'namespace': getNamespace('static', region),
            'locale': 'en_US'
        }, profession_list_uri);

        await cacheSet(PROFESSION_LIST_CACHE, key, result,getStaticTimeWithShift());

        return result;
    }

    /**
     * Get the details for the specified profession.
     * @param profession_id The id of the profession to fetch.
     * @param region The region in which to search.
     */
    async function getBlizProfessionDetail(profession_id: number, region: RegionCode): Promise<BlizzardApi.Profession> {
        const key = `${region}::${profession_id}`;

        if (await cacheCheck(PROFESSION_DETAIL_CACHE, key)) {
            return cacheGet(PROFESSION_DETAIL_CACHE, key);
        }

        const profession_detail_uri = `/data/wow/profession/${profession_id}`;
        const result = <BlizzardApi.Profession>await getBlizzardAPIResponse(region, {
            'namespace': getNamespace('static', region),
            'locale': 'en_US'
        },
            profession_detail_uri);

        await cacheSet(PROFESSION_DETAIL_CACHE, key, result,getStaticTimeWithShift());
        return result;
    }

    async function getBlizConnectedRealmDetail(connected_realm_id: ConnectedRealmID, region: RegionCode): Promise<BlizzardApi.ConnectedRealm> {
        const key = `${region}::${connected_realm_id}`;

        if (await cacheCheck(COMPOSITE_REALM_NAME_CACHE, key)) {
            return cacheGet(COMPOSITE_REALM_NAME_CACHE, key);
        }

        const connected_realm_detail_uri = `/data/wow/connected-realm/${connected_realm_id}`; // skill_tiers.name skill_tiers.id
        const result = <BlizzardApi.ConnectedRealm>await getBlizzardAPIResponse(region, {
            'namespace': getNamespace('dynamic', region),
            'locale': 'en_US'
        },
            connected_realm_detail_uri);

        await cacheSet(COMPOSITE_REALM_NAME_CACHE, key, result, getDynamicTimeWithShift());
        return result;
    }

    /**
     * Fetch the skill tier detail for a profession, the skill tier is
     * what contains things like recipe lists.
     * @param {!number} profession_id The id of the profession.
     * @param {!number} skillTier_id The skill tier id we are interested in.
     * @param {!string} region The region in which to search.
     */
    async function getBlizSkillTierDetail(profession_id: number, skillTier_id: number, region: RegionCode): Promise<BlizzardApi.ProfessionSkillTier> {
        const key = `${region}::${profession_id}::${skillTier_id}`;

        if (await cacheCheck(PROFESSION_SKILL_TIER_DETAILS_CACHE, key)) {
            return cacheGet(PROFESSION_SKILL_TIER_DETAILS_CACHE, key);
        }

        const profession_skill_tier_detail_uri = `/data/wow/profession/${profession_id}/skill-tier/${skillTier_id}`;
        //categories[array].recipes[array].name categories[array].recipes[array].id
        const result = <BlizzardApi.ProfessionSkillTier>await getBlizzardAPIResponse(region, {
            'namespace': getNamespace('static', region),
            'locale': 'en_US'
        },
            profession_skill_tier_detail_uri);

        await cacheSet(PROFESSION_SKILL_TIER_DETAILS_CACHE, key, result, getStaticTimeWithShift());

        return result;
    }

    /**
     * Fetch the details for a recipe, recipe details will include information about
     * required reagents.
     * @param recipe_id The id of the recipe to fetch.
     * @param region The region in which to search.
     */
    async function getBlizRecipeDetail(recipe_id: number, region: RegionCode): Promise<BlizzardApi.Recipe> {
        const key = `${region}::${recipe_id}`;

        if (await cacheCheck(PROFESSION_RECIPE_DETAIL_CACHE, key)) {
            return cacheGet(PROFESSION_RECIPE_DETAIL_CACHE, key);
        }

        const profession_recipe_uri = `/data/wow/recipe/${recipe_id}`;
        //crafted_item.name crafted_item.id / reagents[array].name reagents[array].id reagents[array].quantity

        const result = <BlizzardApi.Recipe>await getBlizzardAPIResponse(region, {
            'namespace': getNamespace('static', region),
            'locale': 'en_US'
        },
            profession_recipe_uri);

        await cacheSet(PROFESSION_RECIPE_DETAIL_CACHE, key, result,getStaticTimeWithShift());

        return result;
    }

    /**
     * Check if an item can be crafted using a given set of professions.
     * @param {!number} item_id The id of the item to check.
     * @param {!Array.<string>}character_professions The list of professions available to the user.
     * @param {!string} region The region in which to search.
     */
    async function checkIsCrafting(item_id: ItemID, character_professions: Array<CharacterProfession>, region: RegionCode): Promise<CraftingStatus> {
        // Check if we've already run this check, and if so return the cached version, otherwise keep on
        const key = `${region}::${item_id}::${JSON.stringify(character_professions)}`;

        if (await cacheCheck(CRAFTABLE_BY_PROFESSION_SET_CACHE, key)) {
            return cacheGet(CRAFTABLE_BY_PROFESSION_SET_CACHE, key);
        }

        const profession_list = await getBlizProfessionsList(region);

        const recipe_options: CraftingStatus = {
            craftable: false,
            recipes: [],
            recipe_ids: []
        };

        // Check if a vendor is mentioned in the item description and if so just short circuit
        const item_detail = await getItemDetails(item_id, region);
        if ('description' in item_detail) {
            if (item_detail.description.includes('vendor')) {
                logger.debug('Skipping vendor recipe');
                await cacheSet(CRAFTABLE_BY_PROFESSION_SET_CACHE, key, recipe_options, getComputedTimeWithShift());
                return recipe_options;
            }
        }

        const profession_result_array: CraftingStatus[] = [];

        // This may be too many concurrent promises
        const character_profession_check_promises: Promise<CraftingStatus>[] = [];
        for (let prof of character_professions) {
            character_profession_check_promises.push(checkProfessionCrafting(profession_list, prof, region, item_id, item_detail));
        }

        (await Promise.all(character_profession_check_promises)).forEach((check) => {
            profession_result_array.push(check);
        });


        // This doesn't spawn nearly as many threads.
        /*
        for (let prof of character_professions) {
            profession_result_array.push(await checkProfessionCrafting(profession_list, prof, region, item_id, item_detail));
        }
        */

        // collate professions
        for (const profession_crafting_check of profession_result_array) {
            recipe_options.recipes = recipe_options.recipes.concat(profession_crafting_check.recipes);
            recipe_options.recipe_ids = recipe_options.recipe_ids.concat(profession_crafting_check.recipe_ids);
            recipe_options.craftable = recipe_options.craftable || profession_crafting_check.craftable;
        }

        await cacheSet(CRAFTABLE_BY_PROFESSION_SET_CACHE, key, recipe_options, getComputedTimeWithShift());
        //{craftable: found_craftable, recipe_id: found_recipe_id, crafting_profession: found_profession};
        return recipe_options;
    }

    function getProfessionId(profession_list: BlizzardApi.ProfessionsIndex, profession_name: string): number {
        const id_src = profession_list.professions.find((item) => {
            return (item.name.localeCompare(profession_name, undefined, { sensitivity: 'accent' }) == 0);
        });
        let id = -1
        if (id_src === undefined) {
            throw new Error('Invalid Profession');
        } else {
            id = id_src.id;
        }
        return id;
    }

    /**
     * Check a single profession to see if an item is craftable by it.
     * @param {Array<object>} profession_list List of all available professions from Blizzard.
     * @param {string} prof The current profession to check.
     * @param {string} region The region in which to search.
     * @param {number} item_id The item id of the item we are checking.
     * @param {object} item_detail Details about the item we are checking.
     */
    async function checkProfessionCrafting(profession_list: BlizzardApi.ProfessionsIndex, prof: CharacterProfession, region: RegionCode, item_id: ItemID, item_detail: BlizzardApi.Item): Promise<CraftingStatus> {
        const cache_key = `${region}:${prof}:${item_id}`;
        if (await cacheCheck(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key)) {
            return cacheGet(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key);
        }

        const profession_recipe_options: CraftingStatus = {
            craftable: false,
            recipes: [],
            recipe_ids: []
        };

        const check_profession_id = getProfessionId(profession_list, prof);

        // Get a list of the crafting levels for the professions
        const profession_detail = await getBlizProfessionDetail(check_profession_id, region);
        const crafting_levels = profession_detail.skill_tiers;

        logger.debug(`Scanning profession: ${profession_detail.name}`);

        // checkProfessionTierCrafting on each crafting level, concurrently.
        await Promise.all(crafting_levels.map((tier) => {
            return checkProfessionTierCrafting(tier, region);
        }));

        await cacheSet(CRAFTABLE_BY_SINGLE_PROFESSION_CACHE, cache_key, profession_recipe_options, getComputedTimeWithShift());

        return profession_recipe_options;

        /**
         * Scan a tier of a given profession to see if it can craft an item.
         * @param skill_tier The tier level to check.
         */
        async function checkProfessionTierCrafting(skill_tier: { name: any; id: any; }, region: RegionCode): Promise<void> {
            let check_scan_tier = skill_tier.name.includes('Shadowlands');
            if (!exclude_before_shadowlands) {
                check_scan_tier = true;
            }
            if (check_scan_tier) {
                logger.debug(`Checking: ${skill_tier.name} for: ${item_id}`);
                // Get a list of all recipes each level can do
                const skill_tier_detail = await getBlizSkillTierDetail(check_profession_id, skill_tier.id, region);

                let checked_categories = 0;
                let recipes_checked = 0;

                if (skill_tier_detail.categories != undefined) {
                    const categories = skill_tier_detail.categories;

                    checked_categories += categories.length;
                    for (const cat of categories) {
                        for (const rec of cat.recipes) {
                            const recipe = await getBlizRecipeDetail(rec.id, region);
                            recipes_checked++;
                            logger.silly(`Check recipe ${recipe.name}`);
                            if (!(recipe.name.includes('Prospect') || recipe.name.includes('Mill'))) {
                                let crafty = false;
                                if (getRecipeCraftedItemID(recipe).has(item_id)) {
                                    crafty = true;
                                }

                                if (!crafty && skill_tier.name.includes('Enchanting') && (cat.name.includes('Enchantments') || cat.name.includes('Echantments'))) {
                                    logger.silly(`Checking if uncraftable item ${item_detail.id} is craftable with a synthetic item-recipe connection.`);
                                    const slot = getSlotName(cat);
                                    const synthetic_item_name = `Enchant ${slot} - ${rec.name}`;
                                    logger.silly(`Generated synthetic item name ${synthetic_item_name}.`);
                                    const synthetic_item_id = await getItemId(region, synthetic_item_name);
                                    logger.silly(`Synthetic item ${synthetic_item_name} has id ${synthetic_item_id}`);
                                    if (synthetic_item_id === item_id) {
                                        crafty = true;
                                        logger.silly(`Synthetic item ${synthetic_item_name} match for ${item_detail.name}.`);
                                    }
                                }
                                else {
                                    logger.silly(`Skipping synthetic for ${crafty} (${!crafty}) ${skill_tier.name} (${skill_tier.name.includes('Enchanting')}) ${cat.name} (${cat.name.includes('Enchantments')}) ${rec.name}`);
                                }

                                if (crafty) {
                                    logger.info(`Found recipe (${recipe.id}): ${recipe.name} for (${item_detail.id}) ${item_detail.name}`);

                                    profession_recipe_options.recipes.push(
                                        {
                                            recipe_id: recipe.id,
                                            crafting_profession: prof
                                        }
                                    );
                                    profession_recipe_options.recipe_ids.push(recipe.id);
                                    profession_recipe_options.craftable = true;
                                }
                            } else {
                                logger.silly(`Skipping Recipe: (${recipe.id}) "${recipe.name}"`);
                            }
                        }
                    }
                } else {
                    logger.debug(`Skill tier ${skill_tier.name} has no categories.`);
                }
                logger.debug(`Checked ${recipes_checked} recipes in ${checked_categories} categories for ${item_id} in ${skill_tier.name}`);
            }
        }
    }

    function getRecipeCraftedItemID(recipe: BlizzardApi.Recipe): Set<ItemID> {
        let item_ids: Set<ItemID> = new Set();
        let found = false;
        if ('alliance_crafted_item' in recipe) {
            item_ids.add(recipe.alliance_crafted_item.id);
            found = true;
        }
        if ('horde_crafted_item' in recipe) {
            item_ids.add(recipe.horde_crafted_item.id);
            found = true;
        }
        if ('crafted_item' in recipe) {
            item_ids.add(recipe.crafted_item.id);
            found = true;
        }
        if (!found) {
            logger.silly(`No crafted id for ${recipe.id}`);
        }

        return item_ids;
    }

    function getSlotName(category: BlizzardApi.ProfessionSkillTier["categories"][number]): string {
        const name = category.name;

        let raw_slot_name = name;

        if (name.includes('Enchantments')) {
            raw_slot_name = name.slice(0, name.lastIndexOf('Enchantments') - 1);
        } else if (name.includes('Echantments')) {
            raw_slot_name = name.slice(0, name.lastIndexOf('Echantments') - 1);
        }

        switch (raw_slot_name) {
            case 'Boot':
                return 'Boots';
            case 'Glove':
                return 'Gloves';
            case 'Chest':
            case 'Cloak':
            case 'Bracer':
            case 'Ring':
            case 'Weapon':
            default:
                return raw_slot_name;
        }
    }

    async function buildCyclicRecipeList(region: RegionCode): Promise<SkillTierCyclicLinks> {

        const profession_list = await getBlizProfessionsList(region);

        const links: SkillTierCyclicLinksBuild = [];

        const id = await getProfessionId(profession_list, 'Enchanting');
        const profz = [await getBlizProfessionDetail(id, region)];
        let counter = 0;
        let profession_counter = 0;

        //for (const prof of profession_list.professions) {
        for (const prof of profz) {
            const last_count = counter;
            logger.debug(`Scanning profession: ${prof.name} for cyclic relationships.`);
            const profession = await getBlizProfessionDetail(prof.id, region);
            if (profession.skill_tiers !== undefined) {
                const st_scan_jobs = profession.skill_tiers.map((st) => {
                    return buildCyclicLinkforSkillTier(st, profession);
                });
                (await Promise.all(st_scan_jobs)).forEach((r: SkillTierCyclicLinksBuild) => {
                    links.push(...r);
                });
            }
            logger.debug(`Scanned ${counter - last_count} new recipes.`);
            profession_counter++;
        }

        logger.debug(`Scanned ${counter} recipes in ${profession_counter} professions`)

        const link_lookup: SkillTierCyclicLinks = {};

        for (const link of links) {
            const item_1 = link[0];
            const item_2 = link[1];

            // Item 1 links
            for (const id_1 of item_1.id) {
                if (link_lookup[id_1] === undefined) {
                    link_lookup[id_1] = [];
                }
                for (const id_2 of item_2.id) {
                    if (id_1 !== id_2) {
                        link_lookup[id_1].push({
                            id: id_2,
                            takes: item_2.quantity,
                            makes: item_1.quantity
                        });
                    }
                }
            }

            // Item 2 links
            for (const id_2 of item_2.id) {
                if (link_lookup[id_2] === undefined) {
                    link_lookup[id_2] = [];
                }
                for (const id_1 of item_1.id) {
                    if (id_2 !== id_1) {
                        link_lookup[id_2].push({
                            id: id_1,
                            takes: item_1.quantity,
                            makes: item_2.quantity
                        });
                    }
                }
            }
        }

        return link_lookup;

        async function buildCyclicLinkforSkillTier(skill_tier: { name: any; id: any; }, profession: BlizzardApi.Profession): Promise<SkillTierCyclicLinksBuild> {
            const cache_key = `${region}::${skill_tier.name}::${profession.id}`;
            if (await cacheCheck(CYCLIC_LINK_CACHE, cache_key)) {
                return cacheGet(CYCLIC_LINK_CACHE, cache_key);
            }
            logger.debug(`Scanning st: ${skill_tier.name}`);
            const checked_set = new Set();
            const found_links: SkillTierCyclicLinksBuild = [];
            const skill_tier_detail = await getBlizSkillTierDetail(profession.id, skill_tier.id, region);
            if (skill_tier_detail.categories !== undefined) {
                for (const sk_category of skill_tier_detail.categories) {
                    for (const sk_recipe of sk_category.recipes) {
                        const recipe = await getBlizRecipeDetail(sk_recipe.id, region);
                        if (!checked_set.has(recipe.id)) {
                            checked_set.add(recipe.id);
                            counter++;
                            if (recipe.reagents !== undefined && recipe.reagents.length === 1) {
                                // Go through them all again
                                for (const sk_recheck_category of skill_tier_detail.categories) {
                                    for (const sk_recheck_recipe of sk_recheck_category.recipes) {
                                        const recheck_recipe = await getBlizRecipeDetail(sk_recheck_recipe.id, region);
                                        if (recheck_recipe.reagents !== undefined && recheck_recipe.reagents.length === 1 && !checked_set.has(recheck_recipe.id)) {
                                            if (getRecipeCraftedItemID(recipe).has(recheck_recipe.reagents[0].reagent.id)) {
                                                if (getRecipeCraftedItemID(recheck_recipe).has(recipe.reagents[0].reagent.id)) {
                                                    logger.debug(`Found cyclic link for ${recipe.name} (${recipe.id}) and ${recheck_recipe.name} (${recheck_recipe.id})`)
                                                    found_links.push(
                                                        [
                                                            {
                                                                id: [...getRecipeCraftedItemID(recipe)],
                                                                quantity: recheck_recipe.reagents[0].quantity
                                                            },
                                                            {
                                                                id: [...getRecipeCraftedItemID(recheck_recipe)],
                                                                quantity: recipe.reagents[0].quantity
                                                            }
                                                        ]);
                                                    checked_set.add(recheck_recipe.id);
                                                }
                                            }
                                        } else {
                                            checked_set.add(recheck_recipe.id);
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
            cacheSet(CYCLIC_LINK_CACHE, cache_key, found_links, getStaticTimeWithShift());
            return found_links;
        }
    }

    /**
     * 
     * @param recipe_id The id of the recipe to check.
     * @param region The region in which to search.
     */
    async function getCraftingRecipe(recipe_id: number, region: RegionCode): Promise<BlizzardApi.Recipe> {
        const recipe = getBlizRecipeDetail(recipe_id, region);
        return recipe;
    }

    /**
     * Fetch a snapshot of the auction house for a given connected realm.
     * @param {!number} server_id The connected realm id to fetch.
     * @param {!string} server_region The region in which the realm exists.
     */
    async function getAuctionHouse(server_id: ConnectedRealmID, server_region: RegionCode): Promise<BlizzardApi.Auctions> {
        // Download the auction house for the server_id
        // If the auction house is older than an hour then remove it from the cached_data.fetched_auction_houses array
        if (await cacheCheck(AUCTION_DATA_CACHE, server_id)) {
            return cacheGet(AUCTION_DATA_CACHE, server_id);
        }

        logger.info('Auction house is out of date, fetching it fresh.')

        const auction_house_fetch_uri = `/data/wow/connected-realm/${server_id}/auctions`;
        const ah = <BlizzardApi.Auctions>await getBlizzardAPIResponse(
            server_region,
            {
                'namespace': getNamespace('dynamic', server_region),
                //'locale': 'en_US'
            },
            auction_house_fetch_uri);

        await cacheSet(AUCTION_DATA_CACHE, server_id, ah, 3600);

        return ah;
    }

    function getNamespace(ns_type: 'dynamic' | 'static', region: RegionCode): string {
        return (`${ns_type}-${region.toLowerCase()}`);
    }

    return Object.freeze({
        getItemId, getConnectedRealmId, getItemDetails, getBlizProfessionsList, getBlizProfessionDetail,
        getBlizSkillTierDetail, getBlizRecipeDetail, checkIsCrafting, getCraftingRecipe, getAuctionHouse,
        getBlizConnectedRealmDetail, buildCyclicRecipeList
    });
}

export { CPCApiHelpers };
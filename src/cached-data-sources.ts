import { promises as fs } from 'fs';
import got from 'got';
import { resolve } from 'path';
import { parentLogger } from './logging.js';

const logger = parentLogger.child({});
let cache_loaded = false;

const bonuses_cache_fn = 'bonuses.json';
const rank_mappings_cache_fn = 'rank-mappings.json';
const shopping_recipe_exclusion_list_fn = 'shopping-recipe-exclusion-list.json'
const data_sources_fn = 'data-sources.json';

let bonuses_cache: BonusesCache;
let rank_mappings_cache: RankMappingsCache;
let shopping_recipe_exclusion_list: ShoppingRecipeExclusionList;

/**
 * Initialize the cache provider.
 */
async function loadCache(init?: StaticCacheConfig): Promise<void> {
    let cache_folder = './cache/';
    if (init !== undefined) {
        if (init.cache_folder !== undefined) {
            cache_folder = init.cache_folder;
        }
    }
    if (!cache_loaded) {
        const data_sources = JSON.parse((await fs.readFile(resolve(cache_folder, data_sources_fn))).toString());

        // static files
        try {
            bonuses_cache = JSON.parse((await fs.readFile(resolve(cache_folder, bonuses_cache_fn))).toString());
        } catch (e) {
            logger.info(`Couldn't find bonuses data, fetching fresh.`);
            const fetched_bonus_data = (await got(data_sources.sources[bonuses_cache_fn].href)).body;
            fs.writeFile(resolve(cache_folder, bonuses_cache_fn), fetched_bonus_data, 'utf8');
            bonuses_cache = JSON.parse(fetched_bonus_data);
        }
        try {
            rank_mappings_cache = JSON.parse((await fs.readFile(resolve(cache_folder, rank_mappings_cache_fn))).toString());
        } catch (e) {
            rank_mappings_cache = {
                available_levels: [190, 210, 225, 235],
                rank_mapping: [0, 1, 2, 3],
            };
        }
        try {
            shopping_recipe_exclusion_list = JSON.parse((await fs.readFile(resolve(cache_folder, shopping_recipe_exclusion_list_fn))).toString());
        } catch (e) {
            shopping_recipe_exclusion_list = {
                exclusions: [],
            };
        }
        cache_loaded = true;
    }
}

async function static_sources(init?: StaticCacheConfig): Promise<StaticSources> {
    await loadCache(init);
    const context = function () { };

    context.bonuses_cache = bonuses_cache;
    context.rank_mappings_cache = rank_mappings_cache;
    context.shopping_recipe_exclusion_list = shopping_recipe_exclusion_list;

    return context;
}

async function CPCCache(database: CPCDB): Promise<CPCCache> {
    const db = await database.getDb('cache');

    /**
 * Check if a key is present in the namespace, optionally checking expiration.
 * @param {!string} namespace The namespace for the key.
 * @param {!string} key The key to check.
 * @param {?number} expiration_period Optionally check if the key has expired.
 */
    async function cacheCheck(namespace: string, key: string | number, expiration_period?: number | undefined): Promise<boolean> {
        //const db = await getDb('cache');
        //logger.profile('cacheGet');
        //const query = 'select namespace, key, value, cached from key_values where namespace = ? and key = ?';
        const query_no_expiration = 'SELECT COUNT(*) AS how_many FROM key_values WHERE namespace = $1 AND key = $2';
        const query_with_expiration = 'SELECT COUNT(*) AS how_many FROM key_values WHERE namespace = $1 AND key = $2 AND (cached + $3) > $4';

        type HowMany = { how_many: number };

        let result: HowMany;

        if (expiration_period !== undefined) {
            const query = query_with_expiration;
            const values = [namespace, key, expiration_period, Date.now()];;
            result = await db.get<HowMany>(query, values);
        } else {
            const query = query_no_expiration;
            const values = [namespace, key];
            result = await db.get<HowMany>(query, values);
        }


        //const result: any = await db.get(query, values);

        let found = false;
        if (result.how_many > 0) {
            found = true;
        }
        //logger.profile('cacheGet');
        return found;
    }

    /**
     * Retrieve a key from the cache.
     * @param {string} namespace The namespace for the key.
     * @param {string} key The key to retrieve.
     */
    async function cacheGet(namespace: string, key: string | number): Promise<any> {
        //const db = await getDb('cache');
        //logger.profile(`cacheGet: ${namespace} -> ${key}`);
        const query = 'SELECT value FROM key_values WHERE namespace = $1 AND key = $2';
        const result: any = await db.get(query, [namespace, key]);
        //const json_data = JSON.parse(result.value);
        //logger.profile(`cacheGet: ${namespace} -> ${key}`);
        return db.db_type === 'pg' ? result.value : JSON.parse(result.value);
    }

    /**
     * Set a key within a namespace to the given value, all other values will be deleted.
     * @param {string} namespace The namespace for the key.
     * @param {string} key The cache key to set.
     * @param {!any} data The value to set the key to.
     */
    async function cacheSet(namespace: string, key: string | number, data: any): Promise<void> {
        //const db = await getDb('cache');
        if (data === undefined) {
            logger.error(`cannot cache undefined to ${namespace} -> ${key}`);
            throw new Error('Cannot cache undefined');
        }
        const cached = Date.now();

        //logger.profile('cacheSet');
        try {
            const query_delete = 'DELETE FROM key_values WHERE namespace = $1 AND key = $2';
            const query_insert = 'INSERT INTO key_values(namespace, key, value, cached) VALUES($1,$2,$3,$4)';

            await db.serialize(
                ['BEGIN TRANSACTION', query_delete, query_insert, 'COMMIT TRANSACTION'],
                [[], [namespace, key], [namespace, key, JSON.stringify(data), cached], []]);
        } catch (e) {
            logger.error('Failed to set up cache value', e);
        }
        //logger.profile('cacheSet');
    }

    async function shutdown() {
        return;
    }

    return Object.freeze({
        cacheCheck,
        cacheGet,
        cacheSet,
        shutdown
    });
}

export {
    static_sources,
    CPCCache
};


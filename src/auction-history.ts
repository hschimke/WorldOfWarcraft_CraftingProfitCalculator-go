import { Logger } from 'winston';
import { CPCApiHelpers } from './blizzard-api-helpers.js';
import { ALL_PROFESSIONS } from './shared-constants.js';
import { getRegionCode } from './getRegionCode.js';

const sql_insert_auction = 'INSERT INTO auctions(item_id, quantity, price, downloaded, connected_realm_id, bonuses, region) VALUES($1,$2,$3,$4,$5,$6,$7)';
const sql_insert_auction_archive = 'INSERT INTO auction_archive(item_id, quantity, summary, downloaded, connected_realm_id, bonuses, region) VALUES($1,$2,$3,$4,$5,$6,$7)';
const sql_insert_item = 'INSERT INTO items(item_id, region, name, craftable, scanned) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING';

async function CPCAuctionHistory(database: CPCDB, logging: Logger, api: CPCApi, cache: CPCCache): Promise<CPCAuctionHistory> {
    const logger = logging;
    const db = await database.getDb('history');
    const db_type = db.db_type;

    const { getAuctionHouse, getConnectedRealmId, checkIsCrafting, getItemId, getItemDetails, getBlizConnectedRealmDetail } = CPCApiHelpers(logging, cache, api);

    async function ingest(region: RegionCode, connected_realm: ConnectedRealmID): Promise<void> {
        logger.info(`Injest job started for ${region}:${connected_realm}`);
        // Get auction house
        const auction_house = await getAuctionHouse(connected_realm, region);
        const downloaded = Date.now();
        // Loop over each auction and add it.
        const items: Record<string, Record<string | number, {
            item_id: ItemID,
            bonus_lists: Array<number>,
            price: number,
            quantity: number
        }>> = {};
        auction_house.auctions.forEach((auction) => {
            const item_id_key = auction.item.id + (('bonus_lists' in auction.item) ? JSON.stringify(auction.item.bonus_lists) : '');
            if (!(item_id_key in items)) {
                items[item_id_key] = {};
            }
            let price = 0;
            const quantity = auction.quantity;
            if (auction.buyout !== undefined) {
                price = auction.buyout;
            } else {
                price = auction.unit_price;
            }
            if (!(price in items[item_id_key])) {
                items[item_id_key][price] = {
                    item_id: auction.item.id,
                    bonus_lists: auction.item.bonus_lists,
                    price: price,
                    quantity: 0,
                };
            }
            items[item_id_key][price].quantity += quantity;
        });

        const item_set: Set<number> = new Set();
        const insert_values_array = [];

        for (const key of Object.keys(items)) {
            for (const pk of Object.keys(items[key])) {
                item_set.add(items[key][pk].item_id);
                //item_id, quantity, price, downloaded, connected_realm_id, bonuses
                insert_values_array.push([items[key][pk].item_id, items[key][pk].quantity, items[key][pk].price, downloaded, connected_realm, JSON.stringify(items[key][pk].bonus_lists), region.toLocaleLowerCase()]);
            }
        }

        const client = await db.getClient();

        type HowMany = { how_many: number };

        await client.query('BEGIN TRANSACTION');

        for (const item of item_set) {
            try {
                await client.query(sql_insert_item, [item, region, null, false, false]);
            } catch (err) {
                logger.error(`Could not save ${item} in region ${region}.`, err);
            }
        }

        await Promise.all(insert_values_array.map((values) => {
            return client.query(sql_insert_auction, values);
        }));

        await client.query('COMMIT TRANSACTION');
        await client.release();

        logger.info(`Injest job finished for ${region}:${connected_realm}`);
    }

    async function getAllBonuses(item: ItemSoftIdentity, region: RegionCode): Promise<GetAllBonusesReturn> {
        logger.debug(`Fetching bonuses for ${item}`);
        const sql = 'SELECT DISTINCT bonuses FROM auctions WHERE item_id = $1';

        let item_id = 0;
        if (typeof item === 'number') {
            item_id = item;
        } else if (Number.isFinite(Number(item))) {
            item_id = Number(item);
        } else {
            item_id = await getItemId(region, item);
            if (item_id < 0) {
                logger.error(`No itemId could be found for ${item}`);
                throw (new Error(`No itemId could be found for ${item}`));
            }
            logger.info(`Found ${item_id} for ${item}`);
        }

        const bonuses: Record<string, string>[] = await db.all(sql, [item_id]);

        logger.debug(`Found ${bonuses.length} bonuses for ${item}`);

        const item_details = await getItemDetails(item_id, region);

        return {
            bonuses: bonuses,
            item: item_details,
        };
    }

    async function fillNItems(fill_count: number = 5): Promise<void> {
        logger.info(`Filling ${fill_count} items with details.`);
        const select_sql = 'SELECT item_id, region FROM items WHERE scanned = false LIMIT $1';
        const update_sql = 'UPDATE items SET name = $1, craftable = $2, scanned = true WHERE item_id = $3 AND region = $4';
        const client = await db.getClient();
        type ItemRow = { item_id: number, region: RegionCode }
        const rows = (await client.query<ItemRow>(select_sql, [fill_count])).rows;
        await client.query('BEGIN TRANSACTION');
        for (const item of rows) {
            try {
                const fetched_item = await getItemDetails(item.item_id, item.region);
                const is_craftable = await checkIsCrafting(item.item_id, ALL_PROFESSIONS, item.region);
                await client.query(update_sql, [fetched_item.name, is_craftable.craftable, item.item_id, item.region]);
                logger.debug(`Updated item: ${item.item_id}:${item.region} with name: '${fetched_item.name}' and craftable: ${is_craftable.craftable}`);
            } catch (e) {
                logger.error(`Issue filling ${item.item_id} in ${item.region}. Skipping`, e);
                await client.query('DELETE FROM items WHERE item_id = $1 AND region = $2', [item.item_id, item.region]);
                logger.error(`DELETED ${item.item_id} in ${item.region} from items table.`);
            }
        }
        await client.query('COMMIT TRANSACTION');
        client.release();
    }

    async function fillNNames(fillCount: number = 5): Promise<void> {
        logger.info(`Filling ${fillCount} unnamed item names.`);
        const select_sql = 'SELECT item_id, region FROM items WHERE name ISNULL ORDER BY item_id DESC LIMIT $1';
        const update_sql = 'UPDATE items SET name = $1 WHERE item_id = $2 AND region = $3';
        const client = await db.getClient();
        type ItemRow = { item_id: number, region: RegionCode }
        const rows = (await client.query<ItemRow>(select_sql, [fillCount])).rows;
        await client.query('BEGIN TRANSACTION');
        for (const item of rows) {
            try {
                const fetched_item = await getItemDetails(item.item_id, item.region);
                await client.query(update_sql, [fetched_item.name, item.item_id, item.region]);
                logger.debug(`Updated item: ${item.item_id}:${item.region} with name: '${fetched_item.name}'`);
            } catch (e) {
                logger.error(`Issue filling ${item.item_id} in ${item.region}. Skipping`, e);
                await client.query('DELETE FROM items WHERE item_id = $1 AND region = $2', [item.item_id, item.region]);
                logger.error(`DELETED ${item.item_id} in ${item.region} from items table.`);
            }
        }
        await client.query('COMMIT TRANSACTION');
        client.release();
    }

    async function getAuctions(item: ItemSoftIdentity, realm: ConnectedRealmSoftIentity, region: RegionCode, bonuses: number[] | string[] | string, start_dtm: number | string | undefined, end_dtm: number | string | undefined): Promise<AuctionSummaryData> {
        logger.debug(`getAuctions(${item}, ${realm}, ${region}, ${bonuses}, ${start_dtm}, ${end_dtm})`);
        //const sql_build = 'SELECT * FROM auctions';
        const sql_archive_build = 'SELECT downloaded, summary FROM auction_archive';
        const sql_build_distinct_dtm = 'SELECT DISTINCT downloaded FROM auctions';
        const sql_build_price_map = 'SELECT price, count(price) AS sales_at_price, sum(quantity) AS quantity_at_price FROM auctions';
        const sql_group_by_price_addin = 'GROUP BY price';
        const sql_build_min = 'SELECT MIN(price) AS min_price FROM auctions';
        const sql_build_max = 'SELECT MAX(price) AS max_price FROM auctions';
        const sql_build_avg = 'SELECT SUM(price*quantity)/SUM(quantity) AS avg_price FROM auctions';
        const sql_build_latest_dtm = 'SELECT MAX(downloaded) AS latest_download FROM auctions';
        const sql_addins = [];
        const value_searches = [];
        if (item !== undefined) {
            // Get specific items
            let item_id = 0;
            if (typeof item === 'number') {
                item_id = item;
            } else if (Number.isFinite(Number(item))) {
                item_id = Number(item);
            } else {
                item_id = await getItemId(region, item);
                if (item_id < 0) {
                    logger.error(`No itemId could be found for ${item}`);
                    throw (new Error(`No itemId could be found for ${item}`));
                }
                logger.info(`Found ${item_id} for ${item}`);
            }
            sql_addins.push(`item_id = ${get_place_marker()}`);
            value_searches.push(item_id);
        } else {
            // All items
        }
        if (realm !== undefined) {
            let server_id = 0;
            if (typeof realm === 'number') {
                server_id = realm;
            } else if (Number.isFinite(Number(realm))) {
                server_id = Number(realm);
            } else {
                server_id = await getConnectedRealmId(realm, region);
                if (server_id < 0) {
                    logger.error(`No connected realm id could be found for ${realm}`);
                    throw (new Error(`No connected realm id could be found for ${realm}`));
                }
                logger.info(`Found ${server_id} for ${realm}`);
            }
            // Get specific realm
            sql_addins.push(`connected_realm_id = ${get_place_marker()}`);
            value_searches.push(server_id);
        } else {
            // All realms
        }
        if (region !== undefined) {
            // Get specific region
            sql_addins.push(`region = ${get_place_marker()}`);
            value_searches.push(region.toLocaleLowerCase());
        } else {
            // All regions
        }
        if (bonuses !== undefined) {
            // Get only with specific bonuses
            if (bonuses === null) {
                sql_addins.push('bonuses IS NULL');
            }
            else if (typeof bonuses === 'string') {
                sql_addins.push(`bonuses = ${get_place_marker()}`);
                value_searches.push(bonuses);
            } else {
                bonuses.forEach((b: string | number) => {
                    if (b !== null && b !== undefined && b !== '') {
                        logger.debug(`Add bonus ${b} in (select json_each.value from json_each(bonuses))`);
                        const json_query = db_type === 'sqlite3' ? `${get_place_marker()} IN (SELECT json_each.value FROM json_each(bonuses))` : `${get_place_marker()} IN (SELECT json_array_elements_text(bonuses::json)::numeric)`
                        sql_addins.push(json_query);
                        value_searches.push(Number(b));
                    }
                });
            }
        } else {
            // any bonuses or none
        }
        if (start_dtm !== undefined) {
            // Include oldest fetch date time
            sql_addins.push(`downloaded >= ${get_place_marker()}`);
            value_searches.push(start_dtm);
        } else {
            // No start fetched date time limit
        }
        if (end_dtm !== undefined) {
            // Include newest fetch date time
            sql_addins.push(`downloaded <= ${get_place_marker()}`);
            value_searches.push(end_dtm);
        } else {
            // No latest fetched date time
        }

        //const run_sql = build_sql_with_addins(sql_build, sql_addins);
        const min_sql = build_sql_with_addins(sql_build_min, sql_addins);
        const max_sql = build_sql_with_addins(sql_build_max, sql_addins);
        const avg_sql = build_sql_with_addins(sql_build_avg, sql_addins);
        const latest_dl_sql = build_sql_with_addins(sql_build_latest_dtm, sql_addins);
        const distinct_download_sql = build_sql_with_addins(sql_build_distinct_dtm, sql_addins);

        const min_dtm_sql = build_sql_with_addins(sql_build_min, [...sql_addins, `downloaded = ${get_place_marker()}`]);
        const max_dtm_sql = build_sql_with_addins(sql_build_max, [...sql_addins, `downloaded = ${get_place_marker()}`]);
        const avg_dtm_sql = build_sql_with_addins(sql_build_avg, [...sql_addins, `downloaded = ${get_place_marker()}`]);
        const price_group_sql = build_sql_with_addins(sql_build_price_map, [...sql_addins, `downloaded = ${get_place_marker()}`]) + ' ' + sql_group_by_price_addin;

        //const client = await db.getClient();
        type MinPrice = { min_price: number };
        type MaxPrice = { max_price: number };
        type AvgPrice = { avg_price: number };
        type LatestDownload = { latest_download: number };
        type Downloaded = { downloaded: number };
        type Summary = { downloaded: number, summary: AuctionPriceSummaryRecord | string };

        const min_value = (await db.get<MinPrice>(min_sql, value_searches)).min_price;
        const max_value = (await db.get<MaxPrice>(max_sql, value_searches)).max_price;
        const avg_value = (await db.get<AvgPrice>(avg_sql, value_searches)).avg_price;
        const latest_dl_value = (await db.get<LatestDownload>(latest_dl_sql, value_searches)).latest_download;

        const price_data_by_download: Record<number, AuctionPriceSummaryRecord> = {};
        for (const row of (await db.all<Downloaded>(distinct_download_sql, value_searches))) {
            price_data_by_download[row.downloaded] = {
                data: await db.all(price_group_sql, [...value_searches, row.downloaded]),
                min_value: (await db.get<MinPrice>(min_dtm_sql, [...value_searches, row.downloaded])).min_price,
                max_value: (await db.get<MaxPrice>(max_dtm_sql, [...value_searches, row.downloaded])).max_price,
                avg_value: (await db.get<AvgPrice>(avg_dtm_sql, [...value_searches, row.downloaded])).avg_price
            };
        }

        // Get archives if they exist
        const archive_fetch_sql = build_sql_with_addins(sql_archive_build, sql_addins);
        const archives = await db.all<Summary>(archive_fetch_sql, value_searches);

        const archived_results: Record<string, Array<AuctionPriceSummaryRecord>> = {};
        logger.debug(`Found ${archives.length} archive rows.`);
        for (const archive of archives) {
            if (!(archive.downloaded in archived_results)) {
                archived_results[archive.downloaded] = [];
            }
            archived_results[archive.downloaded].push((db_type === 'pg' ? archive.summary : JSON.parse(<string>archive.summary)));
        }

        const archive_build = [];

        for (const key of Object.keys(archived_results)) {
            const arch = archived_results[key];

            const arch_build = {
                timestamp: key,
                data: [] as Array<SalesCountSummaryPrice>,
                min_value: Number.MAX_SAFE_INTEGER,
                max_value: Number.MIN_SAFE_INTEGER,
                avg_value: 0,
            };

            const price_link: Record<string, SalesCountSummary> = {};

            for (const a of arch) {

                if (arch_build.min_value > a.min_value) {
                    arch_build.min_value = a.min_value;
                }
                if (arch_build.max_value < a.max_value) {
                    arch_build.max_value = a.max_value;
                }
                arch_build.avg_value += a.avg_value;
                if (a.data !== undefined) {
                    for (const p of a.data) {
                        if (!(p.price in price_link)) {
                            price_link[p.price] = {
                                sales_at_price: 0,
                                quantity_at_price: 0,
                            }
                        }
                        price_link[p.price].sales_at_price += p.sales_at_price;
                        price_link[p.price].quantity_at_price += p.quantity_at_price;
                    }
                }
            }
            arch_build.avg_value = arch_build.avg_value / arch.length;

            Object.keys(price_link).forEach((key) => {
                arch_build.data.push({
                    price: Number(key),
                    sales_at_price: price_link[key].sales_at_price,
                    quantity_at_price: price_link[key].quantity_at_price,
                })
            });

            archive_build.push(arch_build);
        }

        const now_moment = Date.now();
        const spot_summary = await getSpotAuctionSummary(item, realm, region, bonuses);
        if (spot_summary.avg_value !== 0) {
            price_data_by_download[now_moment] = spot_summary;
        }
        const final_latest_value = spot_summary.avg_value === 0 ? latest_dl_value : now_moment;

        logger.debug(`Found max: ${max_value}, min: ${min_value}, avg: ${avg_value}`);

        return {
            min: min_value,
            max: max_value,
            avg: avg_value,
            //latest: latest_dl_value,
            latest: final_latest_value,
            price_map: price_data_by_download,
            archives: archive_build,
        };

        function build_sql_with_addins(base_sql: string, addin_list: Array<string>): string {
            let construct_sql = base_sql;
            if (addin_list.length > 0) {
                construct_sql += ' WHERE ';
                for (const addin of addin_list) {
                    construct_sql += addin;
                    construct_sql += ' AND ';
                }
                construct_sql = construct_sql.slice(0, construct_sql.length - 4);
            }
            return construct_sql;
        }

        function get_place_marker(): string {
            return `$${value_searches.length + 1}`;
        }
    }

    async function archiveAuctions(): Promise<void> {
        const backstep_time_diff = (6.048e+8); // One Week
        //const backstep_time_diff = 1.21e+9; // Two weeks
        const delete_diff = 1.21e+9; // two weeks
        const day_diff = 8.64e+7;
        const backstep_time = Date.now() - backstep_time_diff;

        const sql_get_downloaded_oldest = 'SELECT MIN(downloaded) AS oldest FROM auctions';
        const sql_get_distinct_rows_from_downloaded = 'SELECT DISTINCT item_id, bonuses, connected_realm_id, region FROM auctions WHERE downloaded BETWEEN $1 AND $2';
        const sql_delete_archived_auctions = 'DELETE FROM auctions WHERE downloaded BETWEEN $1 AND $2';

        const sql_price_map = 'SELECT price, count(price) AS sales_at_price, sum(quantity) AS quantity_at_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5 GROUP BY price';
        const sql_min = 'SELECT MIN(price) AS min_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5';
        const sql_max = 'SELECT MAX(price) AS max_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5';
        const sql_avg = 'SELECT SUM(price*quantity)/SUM(quantity) AS avg_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5';

        const client = await db.getClient();

        let count = 0;

        type Oldest = { oldest: number };
        type DistinctRows = { item_id: number, bonuses: string, connected_realm_id: number, region: string };
        type Min = { min_price: number };
        type Max = { max_price: number };
        type Average = { avg_price: number };

        await client.query('BEGIN TRANSACTION', []);

        let running = true;
        while (running) {
            // Get oldest downloaded
            const current_oldest = Number((await client.query<Oldest>(sql_get_downloaded_oldest, [])).rows[0].oldest);
            //console.log((await client.query(sql_get_downloaded_oldest, [])).rows[0].oldest);
            logger.debug(`Current oldest is ${(new Date(current_oldest)).toLocaleString()}`);
            // Check if oldest fits our criteria
            if (current_oldest < backstep_time) {
                // Pick the whole day
                const start_ticks = current_oldest;
                const end_ticks = current_oldest + day_diff;
                logger.info(`Scan between ${(new Date(start_ticks)).toLocaleString()} and ${(new Date(end_ticks)).toLocaleString()}`);
                // Run for that day
                // Get a list of all distinct item/server combinations
                const items = (await client.query<DistinctRows>(sql_get_distinct_rows_from_downloaded, [start_ticks, end_ticks])).rows;
                for (const item of items) {
                    const vals = [item.item_id, item.bonuses, item.connected_realm_id, start_ticks, end_ticks, item.region];

                    // Run the getAuctions command for the combo
                    const summary: AuctionPriceSummaryRecord = {
                        data: (await client.query<SalesCountSummaryPrice>(sql_price_map, vals)).rows,
                        min_value: (await client.query<Min>(sql_min, vals)).rows[0].min_price,
                        max_value: (await client.query<Max>(sql_max, vals)).rows[0].max_price,
                        avg_value: (await client.query<Average>(sql_avg, vals)).rows[0].avg_price
                    }

                    let quantity = 0;
                    if (summary.data !== undefined) {
                        quantity = summary.data.reduce((acc, cur) => {
                            return acc + cur.quantity_at_price;
                        }, 0);
                    }

                    // Add the archive
                    await client.query(sql_insert_auction_archive, [item.item_id, quantity, JSON.stringify(summary), start_ticks, item.connected_realm_id, item.bonuses, item.region]);
                    count++;
                }
                // Delete the archived data
                await client.query(sql_delete_archived_auctions, [start_ticks, end_ticks]);
                // Done
            } else {
                running = false;
                logger.info(`Finished archive task. Archived ${count} records`);
            }
        }

        const delete_backstep = Date.now() - delete_diff;
        const delete_auctions_older = 'DELETE FROM auctions WHERE downloaded < $1';
        const delete_archive_older = 'DELETE FROM auction_archive WHERE downloaded < $1';

        client.query(delete_auctions_older, [delete_backstep]);
        client.query(delete_archive_older, [delete_backstep]);

        await client.query('COMMIT TRANSACTION', []);
        client.release();
    }

    async function getSpotAuctionSummary(item: ItemSoftIdentity, realm: ConnectedRealmSoftIentity, region: RegionCode, bonuses: number[] | string[] | string): Promise<AuctionPriceSummaryRecord> {
        let realm_get = realm;
        if (typeof (realm) === 'string') {
            realm_get = await getConnectedRealmId(realm, region);
        } else if (typeof (realm) === 'number') {
            realm_get = realm;
        } else {
            throw new Error('Realm not a string or number');
        }
        const ah = await getAuctionHouse(realm_get, region);
        logger.debug(`Spot search for item: ${item} and realm ${realm} and region ${region}, with bonuses ${JSON.stringify(bonuses)}`);

        let item_id = item;
        if (typeof (item) === 'string') {
            item_id = await getItemId(region, item);
        }

        const auction_set = ah.auctions.filter((auction) => {
            let found_item = false;
            let found_bonus = false;
            if (auction.item.id == item_id) {
                found_item = true;
                logger.silly(`Found ${auction.item.id}`);
            }

            if (bonuses === null) {
                if (auction.item.bonus_lists === undefined || auction.item.bonus_lists.length === 0) {
                    found_bonus = true;
                    logger.silly(`Found ${auction.item.id} to match null bonus list`);
                }
            } else if (typeof (bonuses) === 'string') {
                const bonus_parse = JSON.parse(bonuses);
                if (Array.isArray(bonus_parse)) {
                    found_bonus = check_bonus(bonus_parse, auction.item.bonus_lists);
                    logger.silly(`String bonus list ${bonuses} returned ${found_bonus} for ${JSON.stringify(auction.item.bonus_lists)}`);
                }
            } else {
                found_bonus = check_bonus(bonuses, auction.item.bonus_lists);
                logger.silly(`Array bonus list ${JSON.stringify(bonuses)} returned ${found_bonus} for ${JSON.stringify(auction.item.bonus_lists)}`);
            }

            return found_bonus && found_item;
        });
        logger.debug(`Found ${auction_set.length} auctions`);

        const return_value: AuctionPriceSummaryRecord = {
            min_value: Number.MAX_SAFE_INTEGER,
            max_value: Number.MIN_SAFE_INTEGER,
            avg_value: 0,
            data: []
        };

        let total_sales = 0;
        let total_price = 0;
        const price_map: Record<number, { quantity: number, sales: number }> = {};

        for (const auction of auction_set) {
            let price = 0;
            const quantity = auction.quantity;
            if (auction.buyout !== undefined) {
                price = auction.buyout;
            } else {
                price = auction.unit_price;
            }

            if (return_value.max_value < price) {
                return_value.max_value = price;
            }
            if (return_value.min_value > price) {
                return_value.min_value = price;
            }
            total_sales += quantity;
            total_price += price * quantity;

            if (price_map[price] === undefined) {
                price_map[price] = {
                    quantity: 0,
                    sales: 0
                }
            }
            price_map[price].quantity += quantity;
            price_map[price].sales += 1
        }

        return_value.avg_value = total_price / total_sales;
        for (const price of Object.keys(price_map)) {
            const p_lookup = Number(price);
            return_value.data?.push(
                {
                    price: p_lookup,
                    quantity_at_price: price_map[p_lookup].quantity,
                    sales_at_price: price_map[p_lookup].sales
                }
            );
        }

        return return_value;

        function check_bonus(bonus_list: number[] | string[], target?: number[]) {
            let found = true;

            // Filter array
            const filtered : string[] | number[] = (bonus_list as any[]).filter(n=>n);
            const numbers = filtered.map(element => Number(element));
            const numbers_only = numbers.filter((number) => {
                return Number.isInteger(number);
            })

            // Take care of undefined targets
            if( target === undefined){
                if(numbers_only.length !== 0){
                    return false;
                }
                return true;
            }

            for( const list_entry of numbers_only ){
                found = found && target.includes(list_entry);
            }
            
            return found;
        };
    }

    async function addRealmToScanList(realm_name: RealmName, realm_region: RegionCode): Promise<void> {
        const sql = 'INSERT INTO realm_scan_list(connected_realm_id,region) VALUES($1,$2)';
        try {
            await db.run(sql, [await getConnectedRealmId(realm_name, realm_region), realm_region.toUpperCase()]);
        } catch (err) {
            logger.error(`Couldn't add ${realm_name} in ${realm_region} to scan realms table.`, err);
        }
    }

    async function removeRealmFromScanList(realm_name: RealmName, realm_region: RegionCode): Promise<void> {
        const sql = 'DELETE FROM realm_scan_list WHERE connected_realm_id = $1 AND region = $2';
        await db.run(sql, [await getConnectedRealmId(realm_name, realm_region), realm_region.toUpperCase()]);
    }

    async function getScanRealms(): Promise<{ realm_names: string, realm_id: ConnectedRealmID, region: RegionCode }[]> {
        const query = 'SELECT connected_realm_id, region FROM realm_scan_list';
        const data = await db.all<{ connected_realm_id: number, region: string }>(query);
        const ret_val: { realm_names: string, realm_id: ConnectedRealmID, region: RegionCode }[] = [];
        for (const row of data) {
            ret_val.push({
                realm_id: row.connected_realm_id,
                region: getRegionCode(row.region),
                realm_names: (await getBlizConnectedRealmDetail(row.connected_realm_id, getRegionCode(row.region))).realms.reduce((prev, rlm) => { return `${prev}, ${rlm.name}` }, '')
            });
        }
        return ret_val;
    }

    async function scanRealms(): Promise<void> {
        type RealmScanListEntry = { region: RegionCode, connected_realm_id: number };
        const getScannableRealms = 'SELECT connected_realm_id, region FROM realm_scan_list';
        const realm_scan_list = await db.all<RealmScanListEntry>(getScannableRealms, []);
        await Promise.all(realm_scan_list.map((realm: RealmScanListEntry) => {
            return ingest(realm.region, realm.connected_realm_id);
        }));
    }

    async function getAllNames(): Promise<string[]> {
        const name_list = await db.all<{ name: string }>('SELECT DISTINCT name FROM items WHERE name NOTNULL');
        return name_list.reduce((prev: string[], curr) => {
            return [...prev, curr.name];
        }, []);
    }

    return Object.freeze({ scanRealms, addRealmToScanList, removeRealmFromScanList, getAuctions, getAllBonuses, archiveAuctions, fillNItems, fillNNames, getScanRealms, getAllNames });
}

export { CPCAuctionHistory };

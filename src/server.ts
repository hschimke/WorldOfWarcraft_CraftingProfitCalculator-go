import cluster from 'cluster';
import compression from 'compression';
import { default as express } from 'express';
import { resolve } from 'path';
import { createClient } from 'redis';
import { CPCApi } from './blizzard-api-call.js';
import { ApiAuthorization } from './blizz_oath.js';
import { CPCCache, static_sources } from './cached-data-sources.js';
import { CPCDb } from './database/database.js';
import { CACHE_DB_FN, CLIENT_ID, CLIENT_SECRET, CLUSTER_SIZE, DATABASE_TYPE, DISABLE_AUCTION_HISTORY, HISTORY_DB_FN, REDIS_URL, SERVER_PORT, USE_REDIS } from './environment_variables.js';
import { getRegionCode } from './getRegionCode.js';
import { parentLogger } from './logging.js';
import { RedisCache } from './redis-cache-provider.js';
import { RunConfiguration } from './RunConfiguration.js';
import { validateProfessions } from './validateProfessions.js';
import { CPCInstance } from './wow_crafting_profits.js';
import { v4 as uuidv4 } from 'uuid';

const logger = parentLogger.child({});

const port = SERVER_PORT;

const include_auction_history: boolean = DISABLE_AUCTION_HISTORY;
const cluster_size: number = CLUSTER_SIZE;

const db_conf: DatabaseConfig = {
    type: DATABASE_TYPE !== undefined ? DATABASE_TYPE : ''
}
if (DATABASE_TYPE === 'sqlite3') {
    db_conf.sqlite3 = {
        cache_fn: CACHE_DB_FN !== undefined ? CACHE_DB_FN : './databases/cache.db',
        auction_fn: HISTORY_DB_FN !== undefined ? HISTORY_DB_FN : './databases/historical_auctions.db'
    };
}

if (cluster.isPrimary) {
    logger.info(`Starting Cluster with ${cluster_size} workers.`);

    for (let i = 0; i < cluster_size; i++) {
        const w = cluster.fork();
        logger.info(`Started worker ${w.id}.`);
    }

    if (include_auction_history) {
        logger.info('Auction History functions enabled');
        await import('./hourly-injest.js');
    }

    let shutting_down = false;

    const shutdown_function = () => {
        logger.info('SIGTERM signal received: closing cluster')
        shutting_down = true;
        if (cluster.workers !== undefined) {
            for (const id in cluster.workers) {
                logger.info(`Destroying worker ${cluster.workers[id]?.process.pid}.`);
                cluster.workers[id]?.kill('SIGTERM');
                logger.info(`Destroyed worker ${id}.`);
            }
        }
    };

    cluster.on('exit', function (worker, code, signal) {
        console.log('worker ' + worker.process.pid + ' died.');
        if (!shutting_down) {
            cluster.fork();
        }
    });

    process.on('SIGTERM', shutdown_function);
    process.on('SIGINT', shutdown_function);
} else {
    const app = express();

    const db = await CPCDb(db_conf, logger);
    const auth = ApiAuthorization(CLIENT_ID, CLIENT_SECRET, logger);
    const api = CPCApi(logger, auth);
    const cache = await (USE_REDIS ? RedisCache() : CPCCache(db));

    app.use(express.urlencoded({ extended: false }));
    app.use(express.json());

    app.use(express.static(resolve('html/build')));

    app.use(compression());

    app.get('/', (req, res) => {
        logger.debug('Homepage requested');
        res.sendFile(resolve('html/build/index.html'));
    });

    app.get('/addon-download', (req, res) => {
        res.download(resolve('html/CraftingProfitCalculator_data.zip'));
    });

    app.get('/healthcheck', (req, res) => {
        logger.debug('Healthcheck OK');
        res.json({ health: 'ok' });
    });

    app.post('/json_output_QUEUED', async (req, res) => {
        const client = createClient({
            url: REDIS_URL
        });

        let json_data: AddonData = { inventory: [], professions: [], realm: { realm_name: '', region_name: '' } };
        if (req.body.addon_data.length > 0) {
            json_data = JSON.parse(req.body.addon_data);
        }
        const uuid = uuidv4();
        await client.connect();

        if (req.body.type == 'custom') {
            logger.debug(`Custom search for item: ${req.body.item_id}, server: ${req.body.server}, region: ${req.body.region}, professions: ${req.body.professions}. JSON DATA: ${json_data.inventory.length}`);
            await client.lPush('cpc-job-queue:web-jobs', JSON.stringify({
                job_id: uuid,
                job_config: {
                    item: req.body.item_id,
                    count: Number(req.body.count),
                    addon_data: {
                        inventory: json_data.inventory,
                        professions: validateProfessions(JSON.parse(req.body.professions)),
                        realm: {
                            realm_name: req.body.server,
                            region_name: req.body.region,
                        },
                    }
                }
            }));
        } else if (req.body.type == 'json') {
            logger.debug('json search');
            await client.lPush('cpc-job-queue:web-jobs', JSON.stringify({
                job_id: uuid,
                job_config: {
                    item: req.body.item_id,
                    count: Number(req.body.count),
                    addon_data: json_data
                }
            }));
            //config = new RunConfiguration(json_data, req.body.item_id, Number(req.body.count));
            logger.debug(`JSON search for item: ${req.body.item_id}, server: ${json_data.realm.realm_name}, region: ${json_data.realm.region_name}, professions: ${json_data.professions}. JSON DATA: ${json_data.inventory.length}`);
        }
        await client.quit();
        res.json({ job_id: uuid });
    });

    app.post('/json_output_CHECK', async (req, res) => {
        const client = createClient({
            url: REDIS_URL
        });
        await client.connect();
        const run_id = req.body.job_id;
        const found = await client.exists(`cpc-job-queue-results:${run_id}`);
        let data : object= { job_id: run_id };
        if(found){
            const SVD = await client.get(`cpc-job-queue-results:${run_id}`);
            if( SVD !== null ){
                await client.del(`cpc-job-queue-results:${run_id}`);
                data = JSON.parse(SVD);
            }
        }
        await client.quit();
        res.json(data);
    });

    app.post('/json_output', (req, res) => {
        let json_data: AddonData = { inventory: [], professions: [], realm: { realm_name: '', region_name: '' } };
        if (req.body.addon_data.length > 0) {
            json_data = JSON.parse(req.body.addon_data);
        }
        let config: RunConfiguration | undefined = undefined;
        if (req.body.type == 'custom') {
            config = new RunConfiguration({
                inventory: json_data.inventory,
                professions: validateProfessions(JSON.parse(req.body.professions)),
                realm: {
                    realm_name: req.body.server,
                    region_name: req.body.region,
                },
            }, req.body.item_id, Number(req.body.count));
            logger.debug(`Custom search for item: ${req.body.item_id}, server: ${req.body.server}, region: ${req.body.region}, professions: ${req.body.professions}. JSON DATA: ${json_data.inventory.length}`);
        } else if (req.body.type == 'json') {
            logger.debug('json search');
            config = new RunConfiguration(json_data, req.body.item_id, Number(req.body.count));
            logger.debug(`JSON search for item: ${config.item_id}, server: ${config.realm_name}, region: ${config.realm_region}, professions: ${config.professions}. JSON DATA: ${json_data.inventory.length}`);
        }


        CPCInstance(logger, cache, api).then((instance) => {
            if (config !== undefined) {
                instance.runWithJSONConfig(config).then((data) => {
                    const { intermediate } = data;
                    res.json(intermediate);
                }).catch((issue) => {
                    logger.info(`Invalid item search`, issue);
                    res.json({ ERROR: 'Item Not Found' });
                });
            }
        });

    });

    if (include_auction_history) {
        logger.info('Auction History functions enabled');
        const { CPCAuctionHistory } = await import('./auction-history.js');
        const ah = await CPCAuctionHistory(db, logger, api, cache);

        app.get('/all_items', (req, res) => {
            const cache_NS = 'AH-FUNCTIONS';
            const cache_KY = 'ALL_ITEMS_NAMES';

            logger.debug('Getting all items');

            const handleNames = (names: string[]) => {
                if (req.query.partial !== undefined && typeof (req.query.partial) === 'string' && req.query.partial !== '') {
                    const partial: string = req.query.partial;
                    logger.debug(`Partial search for all items with "${partial}"`);
                    const filtered_set = names.filter((str) => {
                        return str.toLocaleLowerCase().includes(partial.toLocaleLowerCase());
                    });
                    res.json(filtered_set);
                } else {
                    logger.debug('Returning all unfiltered items.');
                    res.json(names);
                }
            }

            cache.cacheCheck(cache_NS, cache_KY).then((found) => {
                if (found) {
                    logger.debug('Cached all items found.');
                    cache.cacheGet(cache_NS, cache_KY).then(handleNames);
                } else {
                    logger.debug('Getting fresh all items.');
                    ah.getAllNames().then((names) => {
                        cache.cacheSet(cache_NS, cache_KY, names, 4740);
                        handleNames(names);
                    });
                }
            });
        });

        app.get('/scanned_realms', (req, res) => {
            ah.getScanRealms().then((result) => {
                res.json(result);
            })
        });

        app.post('/auction_history', (req, res) => {
            const item: string = req.body.item;
            const realm: string = req.body.realm;
            const region: string = req.body.region;
            const bonuses: string[] = req.body.bonuses;
            const start_dtm: string | undefined = req.body.start_dtm;
            const end_dtm: string | undefined = req.body.end_dtm;

            logger.info(`Request for item: ${item}, realm: ${realm}, region: ${region}, bonuses: ${bonuses}, start_dtm: ${start_dtm}, end_dtm: ${end_dtm}`);
            ah.getAuctions(item, realm, getRegionCode(region), bonuses, start_dtm, end_dtm).then((result: AuctionSummaryData) => {
                logger.debug(`Return auction data`);
                res.json(result);
            }).catch(error => {
                logger.error("Issue getting auctions", error);
                res.json({ ERROR: error });
            });
        });

        app.post('/seen_item_bonuses', (req, res) => {
            const item: string = req.body.item;
            const region: string = req.body.region;

            logger.debug(`Getting seen bonus lists for ${item} in ${region}`);

            if (item === '') {
                res.json({ ERROR: 'empty item' });
            }

            ah.getAllBonuses(item, getRegionCode(region)).then(bonuses => {
                static_sources().then(cache => {
                    const bonuses_cache = cache.bonuses_cache;
                    logger.debug(`Regurning bonus lists for ${item}`);

                    const ilvl_adjusts: Set<number | string> = new Set();
                    const socket_adjusts: Set<number | string> = new Set();
                    const quality_adjusts: Set<number | string> = new Set();
                    const unknown_adjusts: Set<number | string> = new Set();
                    let found_empty_bonuses = false;

                    const b_array = bonuses.bonuses.map(e => {
                        const v: { text: string, parsed: Array<number | string>, reduced: string | undefined } = { text: e.bonuses, parsed: [], reduced: undefined };
                        v.parsed = JSON.parse(e.bonuses);
                        if (v.parsed !== null) {
                            v.reduced = <string>v.parsed.reduce((acc, cur) => {
                                let value = acc;
                                if (bonuses_cache[cur] !== undefined) {
                                    let found = false;
                                    const bonus_link = bonuses_cache[cur];

                                    if (bonus_link.level !== undefined) {
                                        value += `ilevel ${bonuses.item.level + bonus_link.level} `;
                                        found = true;
                                        ilvl_adjusts.add(cur);
                                    }
                                    if (bonus_link.socket !== undefined) {
                                        value += `socketed `;
                                        found = true;
                                        socket_adjusts.add(cur);
                                    }
                                    if (bonus_link.quality !== undefined) {
                                        value += `quality: ${bonuses_cache[cur].quality} `;
                                        found = true;
                                        quality_adjusts.add(cur);
                                    }
                                    if (!found) {
                                        unknown_adjusts.add(cur);
                                    }
                                }
                                return value;
                            }, '');
                        } else {
                            found_empty_bonuses = true;
                        }
                        return v;
                    });

                    const return_value: SeenItemBonusesReturn = {
                        bonuses: bonuses.bonuses,
                        //item: bonuses.item,
                        mapped: b_array,
                        collected: {
                            ilvl: Array.from(ilvl_adjusts).map(
                                (i: number | string) => {
                                    return { id: i, level: bonuses_cache[i].level! + bonuses.item.level }
                                }),
                            socket: Array.from(socket_adjusts).map(
                                (i: number | string) => {
                                    return { id: i, sockets: bonuses_cache[i].socket }
                                }),
                            quality: Array.from(quality_adjusts).map(
                                (i: number | string) => {
                                    return { id: i, quality: bonuses_cache[i].quality }
                                }),
                            unknown: Array.from(unknown_adjusts),
                            empty: found_empty_bonuses,
                        },
                    }

                    //console.log(JSON.stringify(return_value,undefined,2));
                    res.json(return_value);
                })
            }).catch(error => {
                logger.error("Issue getting bonuses", error);
                res.json({ ERROR: error });
            });
        });

        app.post('/bonus_mappings', (req, res) => {
            static_sources().then(
                cache => {
                    res.json(cache.bonuses_cache);
                }
            )
        })
    }

    app.get('*', (req, res) => {
        logger.debug('Unknown route requested');
        res.sendFile(resolve('html/build/index.html'));
    });

    const server = app.listen(port, () => {
        logger.info(`Crafting Profit Calculator running at: http://localhost:${port}`)
    });

    process.on('SIGTERM', () => {
        logger.info('SIGTERM signal received: closing HTTP server')
        db.shutdown().then(() => { cache.shutdown() }).then(() => { server.close() });
    });
}
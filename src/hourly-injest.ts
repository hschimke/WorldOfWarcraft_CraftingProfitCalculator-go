import { CPCAuctionHistory } from './auction-history.js';
import { CPCApi } from './blizzard-api-call.js';
import { ApiAuthorization } from './blizz_oath.js';
import { CPCCache } from './cached-data-sources.js';
import { CPCDb } from './database/database.js';
import { CACHE_DB_FN, CLIENT_ID, CLIENT_SECRET, DATABASE_TYPE, DISABLE_AUCTION_HISTORY, HISTORY_DB_FN, STANDALONE_CONTAINER, USE_REDIS } from './environment_variables.js';
import { parentLogger } from './logging.js';
import { RedisCache } from './redis-cache-provider.js';


const logger = parentLogger.child({});

const server_mode = STANDALONE_CONTAINER;
const include_auction_history: boolean = DISABLE_AUCTION_HISTORY;

let standalone_container_abc: NodeJS.Timeout | undefined = undefined;

//await addRealmToScanList('hyjal','us');

async function job(ah: CPCAuctionHistory) {
    logger.info('Starting hourly injest job.');

    await ah.scanRealms();
    await ah.fillNItems(20);
    if ((new Date()).getHours() === 4) {
        logger.info('Performing daily archive.');
        await ah.archiveAuctions();
    }
    logger.info('Finished hourly injest job.');
    /*ah.scanRealms().then(() => {
        return ah.fillNItems(20);
    }).then(() => {
        if ((new Date()).getHours() === 4) {
            logger.info('Performing daily archive.');
            ah.archiveAuctions();
        }
    }).finally(() => {
        logger.info('Finished hourly injest job.');
    })*/
}

async function fillNames(ah: CPCAuctionHistory) {
    return ah.fillNNames(100);
}


if (include_auction_history) {
    switch (server_mode) {
        case 'hourly':
            {
                logger.info('Started in default mode. Running job and exiting.');
                const db_conf: DatabaseConfig = {
                    type: DATABASE_TYPE !== undefined ? DATABASE_TYPE : ''
                }
                if (DATABASE_TYPE === 'sqlite3') {
                    db_conf.sqlite3 = {
                        cache_fn: CACHE_DB_FN !== undefined ? CACHE_DB_FN : './databases/cache.db',
                        auction_fn: HISTORY_DB_FN !== undefined ? HISTORY_DB_FN : './databases/historical_auctions.db'
                    };
                }
                const db = await CPCDb(db_conf, logger);
                const auth = ApiAuthorization(CLIENT_ID, CLIENT_SECRET, logger);
                const api = CPCApi(logger, auth);
                const cache = await (USE_REDIS ? RedisCache() : CPCCache(db));
                const ah = await CPCAuctionHistory(db, logger, api, cache);
                await job(ah);
                await fillNames(ah);
                api.shutdownApiManager();
                await cache.shutdown();
                await db.shutdown();
                process.exit(0);
                break;
            }
        case 'worker':
            {
                logger.info('Started as a worker thread, actions will be as if standalone but no server is running elsewhere.');
            }
        case 'standalone':
            {
                logger.info('Started in standalone container mode. Scheduling hourly job.');
                logger.info('Started in standalone container mode. Scheduling name fetch job.')
                const db_conf: DatabaseConfig = {
                    type: DATABASE_TYPE !== undefined ? DATABASE_TYPE : ''
                }
                if (DATABASE_TYPE === 'sqlite3') {
                    db_conf.sqlite3 = {
                        cache_fn: CACHE_DB_FN !== undefined ? CACHE_DB_FN : './databases/cache.db',
                        auction_fn: HISTORY_DB_FN !== undefined ? HISTORY_DB_FN : './databases/historical_auctions.db'
                    };
                }
                const db = await CPCDb(db_conf, logger);
                const auth = ApiAuthorization(CLIENT_ID, CLIENT_SECRET, logger);
                const api = CPCApi(logger, auth);
                const cache = await (USE_REDIS ? RedisCache() : CPCCache(db));
                const ah = await CPCAuctionHistory(db, logger, api, cache);

                const standalone_container_name_fetch = setInterval(() => { fillNames(ah) }, 300000);
                standalone_container_name_fetch.unref();

                standalone_container_abc = setInterval(() => {
                    if (((new Date()).getHours() % 3) === 0) { // run only on hours divisible by 3
                        job(ah);
                    }
                }, 3.6e+6);
                standalone_container_abc.unref();
                break;
            }
        case 'normal':
        default:
            logger.info('Started in normal mode taking no action.');
            break;
    }
}
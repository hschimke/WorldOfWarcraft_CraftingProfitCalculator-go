import { commandOptions, createClient } from 'redis';
import { setTimeout } from 'timers/promises';
import { config } from 'winston';
import { CPCApi } from './blizzard-api-call.js';
import { ApiAuthorization } from './blizz_oath.js';
import { CPCCache } from './cached-data-sources.js';
import { CPCDb } from './database/database.js';
import { CACHE_DB_FN, CLIENT_ID, CLIENT_SECRET, DATABASE_TYPE, HISTORY_DB_FN, REDIS_URL, USE_REDIS } from './environment_variables.js';
import { parentLogger } from './logging.js';
import { RedisCache } from './redis-cache-provider.js';
import { RunConfiguration } from './RunConfiguration.js';
import { CPCInstance } from './wow_crafting_profits.js';

const logger = parentLogger.child({});

logger.info('Starting cpc-job-worker');

const client = createClient({
    url: REDIS_URL
});

let running = true;

process.on('SIGINT', () => {
    running = false;
});

await client.connect();

const db_conf: DatabaseConfig = {
    type: DATABASE_TYPE !== undefined ? DATABASE_TYPE : ''
}
if (DATABASE_TYPE === 'sqlite3') {
    db_conf.sqlite3 = {
        cache_fn: CACHE_DB_FN !== undefined ? CACHE_DB_FN : './databases/cache.db',
        auction_fn: HISTORY_DB_FN !== undefined ? HISTORY_DB_FN : './databases/historical_auctions.db'
    };
}

interface job_run {
    job_id: string;
    job_config: {
        item: ItemSoftIdentity;
        count: number;
        addon_data: AddonData;
    };
};

const db = await CPCDb(db_conf, logger);
const auth = ApiAuthorization(CLIENT_ID, CLIENT_SECRET, logger);
const api = CPCApi(logger, auth);
const cache = await (USE_REDIS ? RedisCache() : CPCCache(db));

while (running) {
    logger.debug('Trying to get job');
    const raw_job_data = await client.brPop(commandOptions({ isolated: true }),'cpc-job-queue:web-jobs', 15);
    if (raw_job_data !== null) {
        const job_data = JSON.parse(raw_job_data.element) as job_run;
        const run_id = job_data.job_id;
        const run_config = job_data.job_config;
        logger.info(`Got new job with id ${run_id}`, run_config);
        const config = new RunConfiguration(run_config.addon_data, run_config.item, Number(run_config.count));
        const instance = await CPCInstance(logger, cache, api);
        if (config !== undefined) {
            instance.runWithJSONConfig(config).then((data) => {
                const { intermediate } = data;
                client.setEx(`cpc-job-queue-results:${run_id}`, 3600, JSON.stringify(intermediate)); // one hour
            }).catch((issue) => {
                logger.info(`Invalid item search`, issue);
                client.setEx(`cpc-job-queue-results:${run_id}`, 3600, JSON.stringify({ ERROR: 'Item Not Found' })); // one hour
            });
        }
    }
}

await client.quit();
await cache.shutdown();
await api.shutdownApiManager();
await db.shutdown();
logger.info('Stopping cpc-job-worker');
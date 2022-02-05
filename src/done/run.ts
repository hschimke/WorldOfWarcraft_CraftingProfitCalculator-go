import yargs from 'yargs';
import { hideBin } from 'yargs/helpers';
import { CPCApi } from './blizzard-api-call.js';
import { ApiAuthorization } from './blizz_oath.js';
import { CPCCache } from './cached-data-sources.js';
import { RedisCache } from './redis-cache-provider.js';
import { CPCDb } from './database/database.js';
import { parentLogger } from './logging.js';
import { RunConfiguration } from './RunConfiguration.js';
import { validateProfessions } from "./validateProfessions.js";
import { CPCInstance } from './wow_crafting_profits.js';
import { CACHE_DB_FN, CLIENT_ID, CLIENT_SECRET, DATABASE_TYPE, HISTORY_DB_FN, USE_REDIS } from './environment_variables.js';

//blacksmithing
//const test_item = 171414;
//tailoring
//const test_item = 173244;
//simple wood
//const test_item = 4470;
// Flask
//const test_item = 171276;
// Bracers
//173249

const parser = yargs(hideBin(process.argv))
    .option('region', {
        description: 'Region',
        alias: 'r',
        type: 'string',
        default: 'US',
    })
    .option('server', {
        description: 'Server',
        alias: 's',
        type: 'string',
        default: 'Hyjal',
    })
    .option('profession', {
        description: 'Profession',
        alias: 'p',
        type: 'string',
        default: ['Jewelcrafting', 'Tailoring', 'Alchemy', 'Herbalism', 'Inscription', 'Enchanting', 'Blacksmithing', 'Mining', 'Engineering', 'Leatherworking', 'Skinning', 'Cooking'],
    })
    .option('item', {
        description: 'Item',
        alias: 'i',
        type: 'string',
        default: '171276',
    })
    .option('count', {
        description: 'How many of the main item to build',
        alias: 'c',
        type: 'number',
        default: 1
    })
    .option('json_data', {
        description: 'JSON configuration data',
        alias: 'j',
        type: 'string',
        default: '',
    })
    .command('json', 'Use JSON to configure region, realm, and professions')
    .help()
    .alias('help', 'h');

let character_config_json: AddonData = { inventory: [], realm: { realm_name: '', region_name: '' }, professions: [] };

const argv = await parser.argv;

let region = argv['region'];
let server = argv.server;
let professions = argv.profession;
let item = argv.item;
let required = argv.count;

try {
    character_config_json = JSON.parse(argv.json_data);
    let good = true;
    let has_inventory = true;
    let has_professions = true;
    let has_realm = true;
    if (!character_config_json.hasOwnProperty('inventory')) {
        good = false;
        has_inventory = false;
    }
    if (!character_config_json.hasOwnProperty('professions')) {
        good = false;
        has_professions = false;
    }
    if (!character_config_json.hasOwnProperty('realm')) {
        good = false;
        has_realm = false;
    }

    if (good) {
        if (argv.json) {
            region = character_config_json.realm.region_name;
            server = character_config_json.realm.realm_name;

            professions = Array.from(new Set(character_config_json.professions));
        }
    }
} catch (e) {
    console.log('JSON character input cannot be parsed.')
}
const config = new RunConfiguration({
    inventory: character_config_json.inventory,
    professions: validateProfessions(professions),
    realm: {
        realm_name: server,
        region_name: region,
    },
}, item, required);

const db_conf: DatabaseConfig = {
    type: DATABASE_TYPE !== undefined ? DATABASE_TYPE : ''
}
if (DATABASE_TYPE === 'sqlite3') {
    db_conf.sqlite3 = {
        cache_fn: CACHE_DB_FN !== undefined ? CACHE_DB_FN : './databases/cache.db',
        auction_fn: HISTORY_DB_FN !== undefined ? HISTORY_DB_FN : './databases/historical_auctions.db'
    };
}
const log = parentLogger.child({})

/*log.add(new winston.transports.Stream({
    stream: createWriteStream('raw_logger'),
    format: winston.format.simple(),
        level: 'silly',
    }
));*/

const db = await CPCDb(db_conf, log);
const auth = ApiAuthorization(CLIENT_ID, CLIENT_SECRET, log);
const api = CPCApi(log, auth);
//const cache = await CPCCache(db);
//const cache = await RedisCache();
const cache = await (USE_REDIS ? RedisCache() : CPCCache(db));
const inst = await CPCInstance(log, cache, api);

await inst.cliRun(config)
await db.shutdown();
await cache.shutdown();
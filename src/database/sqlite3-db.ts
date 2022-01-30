import { default as sqlite3 } from 'sqlite3';
import { Logger } from 'winston';
import { sqlToString } from './shared.js';

const pragma_sync = 'PRAGMA synchronous = normal';
const pragma_journal = 'PRAGMA journal_mode=WAL';

const sql_create_system_table = 'CREATE TABLE IF NOT EXISTS system (key TEXT, value JSON, PRIMARY KEY (key))';

const sql_create_item_table_sq3 = 'CREATE TABLE IF NOT EXISTS auctions (item_id INTEGER, bonuses TEXT, quantity INTEGER, price INTEGER, downloaded INTEGER, connected_realm_id INTEGER, region TEXT)';
const sql_create_auctions_index_sq3 = 'CREATE INDEX IF NOT EXISTS auctions_index ON auctions (item_id, bonuses, quantity, price, downloaded, connected_realm_id, region)';
const sql_create_items_table_sq3 = 'CREATE TABLE IF NOT EXISTS items (item_id INTEGER, region TEXT, name TEXT, craftable INTEGER, scanned INTEGER, PRIMARY KEY (item_id,region))';
const sql_create_realm_scan_table_sq3 = 'CREATE TABLE IF NOT EXISTS realm_scan_list (connected_realm_id INTEGER, region TEXT, PRIMARY KEY (connected_realm_id,region))';
const sql_create_archive_table_sq3 = 'CREATE TABLE IF NOT EXISTS auction_archive (item_id INTEGER, bonuses TEXT, quantity INTEGER, summary TEXT, downloaded INTEGER, connected_realm_id INTEGER, region TEXT)';
const sql_create_auction_archive_index_sq3 = 'CREATE INDEX IF NOT EXISTS auction_archive_index ON auction_archive (item_id, bonuses, downloaded, connected_realm_id, region)';
const sql_create_items_name_ind_sq3 = 'CREATE INDEX IF NOT EXISTS items_name_index on items (name)';

const history_sql_run_at_open_sq3 = [
    pragma_sync,
    pragma_journal,
    sql_create_item_table_sq3,
    sql_create_items_table_sq3,
    sql_create_realm_scan_table_sq3,
    sql_create_auctions_index_sq3,
    sql_create_archive_table_sq3,
    sql_create_auction_archive_index_sq3,
    sql_create_system_table,
    sql_create_items_name_ind_sq3
];

const sql_create_cache_table_sq3 = 'CREATE TABLE IF NOT EXISTS key_values (namespace TEXT, key TEXT, value TEXT, cached INTEGER, PRIMARY KEY (namespace,key))';

const cache_sql_run_at_open_sq3 = [
    sql_create_cache_table_sq3,
    pragma_sync,
    pragma_journal];



function CPC_SQLITE3_DB(config: DatabaseConfig, logging: Logger) {
    const logger = logging;
    const dbs = new Map<string, sqlite3.Database>();
    let dbs_open: boolean;

    async function start(): Promise<void> {
        if (!dbs_open) {
            logger.debug('Opening DBs');
            const cache_fn = config.sqlite3 !== undefined ? config.sqlite3.cache_fn : '';
            const history_fn = config.sqlite3 !== undefined ? config.sqlite3.auction_fn : '';
            dbs.set('cache', new sqlite3.Database(cache_fn, sqlite3.OPEN_CREATE | sqlite3.OPEN_READWRITE));
            dbs.set('history', new sqlite3.Database(history_fn, sqlite3.OPEN_CREATE | sqlite3.OPEN_READWRITE));

            dbs.get('history')?.serialize(() => {
                for (const query of history_sql_run_at_open_sq3) {
                    dbs.get('history')?.run(query);
                }
            });

            dbs.get('cache')?.serialize(() => {
                for (const query of cache_sql_run_at_open_sq3) {
                    dbs.get('cache')?.run(query);
                }
            });

            dbs_open = true;
        }
    }

    async function shutdown(): Promise<void> {
        logger.info('Closing DB connection');
        dbs_open = false;
        let closes = [];
        for (const [key, value] of dbs) {
            closes.push(new Promise((accept,reject)=>{
                logger.info(`Close DB ${key}`);
                value.run('PRAGMA optimize', (err: any) => {
                    value.close(()=>{
                        accept(1);
                    });
                    if(err){
                        reject(err);
                    }
                });
            }));
        }
        await Promise.all(closes);
    }

    async function getDb(db_name: string): Promise<DatabaseManagerFunction> {
        await start();
        const context: Sqlite3DatabaseManagerFunction = function () { };
        context.db_type = 'sqlite3';
        context.db = dbs.get(db_name);
        context.get = function (query, values?) {
            logger.silly(sqlToString(query, values));
            return new Promise((accept, reject) => {
                this.db.get(query, values, (err: any, row: any) => {
                    if (err) {
                        logger.error(`Issue running query '${query}' and values ${values}`, err);
                        reject();
                    }
                    accept(row);
                })
            });
        };
        context.run = function (query, values?) {
            logger.silly(sqlToString(query, values));
            return new Promise<void>((accept, reject) => {
                this.db.run(query, values, (err: any) => {
                    if (err) {
                        logger.error(`Issue running query '${query}' and values ${values}`, err);
                        reject();
                    } else {
                        accept();
                    }
                })
            });
        };
        context.all = function (query, values?) {
            logger.silly(sqlToString(query, values));
            return new Promise((accept, reject) => {
                this.db.all(query, values, (err: any, rows: any[]) => {
                    if (err) {
                        logger.error(`Issue running query '${query}' and values ${values}`, err);
                        reject();
                    }
                    accept(rows);
                })
            });
        };
        context.serialize = function (queries, value_sets) {
            return new Promise<void>((accept, reject) => {
                this.db.serialize(() => {
                    try {
                        for (let i = 0; i < queries.length; i++) {
                            if (value_sets !== undefined)
                                logger.silly(sqlToString(queries[i], value_sets[i]));
                            this.db.run(queries[i], value_sets[i], (err: any) => {
                                if (err) {
                                    logger.error(`Issue running query '${queries[i]}' and values ${value_sets[i]}`, err);
                                    reject();
                                }
                            });
                        }
                        accept();
                    } catch (e) {
                        logger.error('serialize failed', { q: queries, v: value_sets });
                        reject(e);
                    }
                })
            });
        };
        context.query = function (query, values?) {
            return new Promise((acc, reg) => {
                context.all(query, values).then(((result: any) => {
                    acc({ rows: [...result] });
                }))
            });
        };
        context.getClient = async function () {
            const f = function () { };
            f.release = async function () { };
            f.query = async function (query: string, values?: any) {
                const result = <Array<any>>(await context.all(query, values));
                return { rows: [...result] };
            };
            return f;
        };

        return context;
    }

    return Object.freeze({
        getDb,
        shutdown
    });
}

export { CPC_SQLITE3_DB }
import { default as pg } from 'pg';
import { Logger } from 'winston';
import { sqlToString } from './shared.js';
const { Pool } = pg;

const sql_create_system_table = 'CREATE TABLE IF NOT EXISTS system (key TEXT, value JSON, PRIMARY KEY (key))';

const sql_create_item_table_pg = 'CREATE TABLE IF NOT EXISTS auctions (item_id NUMERIC, bonuses TEXT, quantity NUMERIC, price NUMERIC, downloaded NUMERIC, connected_realm_id NUMERIC, region TEXT)';
const sql_create_auctions_index_pg = 'CREATE INDEX IF NOT EXISTS auctions_index ON auctions (item_id, bonuses, quantity, price, downloaded, connected_realm_id, region)';
const sql_create_items_table_pg = 'CREATE TABLE IF NOT EXISTS items (item_id NUMERIC, region TEXT, name TEXT, craftable BOOLEAN, scanned BOOLEAN, PRIMARY KEY (item_id,region))';
const sql_create_items_name_ind_pg = 'CREATE INDEX IF NOT EXISTS items_name_index on items (name)';
const sql_create_realm_scan_table_pg = 'CREATE TABLE IF NOT EXISTS realm_scan_list (connected_realm_id NUMERIC, region TEXT, PRIMARY KEY (connected_realm_id,region))';
const sql_create_archive_table_pg = 'CREATE TABLE IF NOT EXISTS auction_archive (item_id NUMERIC, bonuses TEXT, quantity NUMERIC, summary JSON, downloaded NUMERIC, connected_realm_id NUMERIC, region TEXT)';
const sql_create_auction_archive_index_pg = 'CREATE INDEX IF NOT EXISTS auction_archive_index ON auction_archive (item_id, bonuses, downloaded, connected_realm_id, region)';

const history_sql_run_at_open_pg = [
    sql_create_item_table_pg,
    sql_create_items_table_pg,
    sql_create_realm_scan_table_pg,
    sql_create_archive_table_pg,
    sql_create_auctions_index_pg,
    sql_create_auction_archive_index_pg,
    sql_create_system_table,
    sql_create_items_name_ind_pg
];

const sql_create_cache_table_pg = 'CREATE TABLE IF NOT EXISTS key_values (namespace TEXT, key TEXT, value JSON, cached NUMERIC, PRIMARY KEY (namespace,key))';

const cache_sql_run_at_open_pg = [
    sql_create_cache_table_pg];

function CPC_PG_DB(config: DatabaseConfig, logging: Logger) {
    const logger = logging;
    let l_pool: pg.Pool;
    let dbs_open: boolean;

    async function start(): Promise<void> {
        if (!dbs_open) {
            l_pool = new Pool();
            const client = await l_pool.connect();

            for (const query of history_sql_run_at_open_pg) {
                await client.query(query);
            }
            for (const query of cache_sql_run_at_open_pg) {
                await client.query(query);
            }
            await client.release();

            dbs_open = true;
        }
    }

    async function shutdown(): Promise<void> {
        logger.info('Closing DB connection');
        dbs_open = false;
        await l_pool.end();
        logger.info('Database connection closed');
    }

    async function getDb(db_name: string): Promise<DatabaseManagerFunction> {
        await start();
        const context: PostgresDatabaseManagerFunction = function () { };
        context.pool = l_pool;
        context.db_type = 'pg';
        context.serialize = async function (queries, value_sets) {
            const client = await this.pool.connect();
            for (let i = 0; i < queries.length; i++) {
                logger.silly(sqlToString(queries[i], value_sets[i]));
                await client.query(queries[i], value_sets[i]);
            }
            await client.release();
        };
        context.getClient = async function () {
            const client = await this.pool.connect();
            return client;
        };
        context.query = async function (query, values?) {
            logger.silly(sqlToString(query, values));
            const res = await this.pool.query(query, values);
            return res;
        };
        context.get = async function (query, values?) {
            const res = await this.query(query, values);
            return res.rows[0] as any;
        }
        context.all = async function (query, values?) {
            const result = await this.query(query, values);
            return result.rows as any;
        }
        context.run = async function (query, values?) {
            logger.silly(sqlToString(query, values));
            await this.pool.query(query, values);
        }

        return context;
    }

    return Object.freeze({
        getDb,
        shutdown
    })
}

export { CPC_PG_DB }
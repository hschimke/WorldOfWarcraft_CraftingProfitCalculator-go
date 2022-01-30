//const server_mode = process.env.STANDALONE_CONTAINER === undefined ? 'normal' : process.env.STANDALONE_CONTAINER;
export type STANDALONE_CONTAINER_TYPE = "normal" | "hourly" | "worker" | "standalone";
if (!isStandAloneContainerOption(process.env.STANDALONE_CONTAINER)) {
    throw new Error("STANDALONE_CONTAINER undefined");
}
const STANDALONE_CONTAINER: STANDALONE_CONTAINER_TYPE = process.env.STANDALONE_CONTAINER;

//const include_auction_history: boolean = process.env.DISABLE_AUCTION_HISTORY !== undefined && process.env.DISABLE_AUCTION_HISTORY === 'true' ? false : true;
export type DISABLE_AUCTION_HISTORY_TYPE = boolean;
const DISABLE_AUCTION_HISTORY: DISABLE_AUCTION_HISTORY_TYPE = process.env.DISABLE_AUCTION_HISTORY !== undefined && process.env.DISABLE_AUCTION_HISTORY === 'true' ? false : true;

export type DATABASE_TYPE_TYPE = "sqlite3" | "pg";
if (!isDatabaseTypeOption(process.env.DATABASE_TYPE)) {
    throw new Error("DATABASE_TYPE undefined");
}
const DATABASE_TYPE: DATABASE_TYPE_TYPE = process.env.DATABASE_TYPE;

export type CACHE_DB_FN_TYPE = string | undefined;
const CACHE_DB_FN: CACHE_DB_FN_TYPE = process.env.CACHE_DB_FN;

export type HISTORY_DB_FN_TYPE = string | undefined;
const HISTORY_DB_FN: HISTORY_DB_FN_TYPE = process.env.HISTORY_DB_FN;

export type CLIENT_ID_TYPE = string;
if (process.env.CLIENT_ID === undefined) {
    throw new Error("CLIENT_ID cannot be undefined.");
}
const CLIENT_ID: CLIENT_ID_TYPE = process.env.CLIENT_ID;

export type CLIENT_SECRET_TYPE = string;
if (process.env.CLIENT_SECRET === undefined) {
    throw new Error("CLIENT_SECRET cannot be undefined.");
}
const CLIENT_SECRET: CLIENT_SECRET_TYPE = process.env.CLIENT_SECRET;

export type USE_REDIS_TYPE = boolean;
const USE_REDIS: USE_REDIS_TYPE = ((process.env.USE_REDIS === undefined) || (process.env.USE_REDIS === 'false')) ? false : true;

export type LOG_LEVEL_TYPE = string;
const LOG_LEVEL: LOG_LEVEL_TYPE = (process.env.LOG_LEVEL !== undefined) ? process.env.LOG_LEVEL : 'info';

export type NODE_ENV_TYPE = string;
const NODE_ENV: NODE_ENV_TYPE = (process.env.NODE_ENV !== undefined) ? process.env.NODE_ENV : 'development';

export type DOCKERIZED_TYPE = boolean;
const DOCKERIZED: DOCKERIZED_TYPE = (process.env.DOCKERIZED === undefined || process.env.DOCKERIZED === 'false') ? false : true;

export type REDIS_URL_TYPE = string | undefined;
const REDIS_URL: REDIS_URL_TYPE = process.env.REDIS_URL;

export type SERVER_PORT_TYPE = string | number;
if (typeof process.env.SERVER_PORT !== "number" && typeof process.env.SERVER_PORT !== "string") {
    throw new Error("SERVER_PORT must be a number or string.");
}
const SERVER_PORT: SERVER_PORT_TYPE = process.env.SERVER_PORT;

export type CLUSTER_SIZE_TYPE = number;
const CLUSTER_SIZE: CLUSTER_SIZE_TYPE = process.env.CLUSTER_SIZE !== undefined && Number.isSafeInteger(Number(process.env.CLUSTER_SIZE)) ? Number(process.env.CLUSTER_SIZE) : 1;

function isStandAloneContainerOption(candidate: unknown): candidate is STANDALONE_CONTAINER_TYPE {
    if (typeof candidate !== 'string') {
        return false;
    }
    switch (candidate) {
        case 'normal':
        case 'hourly':
        case 'worker':
        case 'standalone':
            return true;
        default:
            return false;
    }
}

function isDatabaseTypeOption(candidate: unknown): candidate is DATABASE_TYPE_TYPE {
    if (typeof candidate !== 'string') {
        return false;
    }
    switch (candidate) {
        case 'sqlite3':
        case 'pg':
            return true;
        default:
            return false;
    }
}

export {
    STANDALONE_CONTAINER,
    DISABLE_AUCTION_HISTORY,
    DATABASE_TYPE,
    CACHE_DB_FN,
    HISTORY_DB_FN,
    CLIENT_ID,
    CLIENT_SECRET,
    USE_REDIS,
    LOG_LEVEL,
    NODE_ENV,
    DOCKERIZED,
    REDIS_URL,
    SERVER_PORT,
    CLUSTER_SIZE
}
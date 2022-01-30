import { Logger } from 'winston';

async function CPCDb(config: DatabaseConfig, logging: Logger): Promise<CPCDB> {
    const db_type = config.type;

    const logger = logging;

    /*
    import winston from 'winston';
    logger.add(new winston.transports.Console({
        format: winston.format.simple(),
        level: 'silly',
    }));*/

    logger.info(`Using ${db_type} database methods.`);

    switch (db_type) {
        case 'pg':
            const { CPC_PG_DB } = await import('./postgres-db.js');
            return CPC_PG_DB(config, logger);
        case 'sqlite3':
            const { CPC_SQLITE3_DB } = await import('./sqlite3-db.js');
            return CPC_SQLITE3_DB(config, logger);
    }
    throw new Error(`UNDEFINED DATABASE TYPE: ${db_type}`);
}

export { CPCDb };

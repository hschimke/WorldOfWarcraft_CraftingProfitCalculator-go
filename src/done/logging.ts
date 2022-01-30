import { default as winston } from 'winston';
import { DOCKERIZED, LOG_LEVEL, NODE_ENV } from './environment_variables.js';

const lowest_level_to_report: string = LOG_LEVEL;

const parentLogger = winston.createLogger({
    level: lowest_level_to_report,
    format: winston.format.json(),
    //defaultMeta: { service: 'user-service' },
    transports: [
        //
        // - Write all logs with level `error` and below to `error.log`
        // - Write all logs with level `info` and below to `combined.log`
        //
        //new winston.transports.File({ filename: 'error.log', level: 'error' }),
        //new winston.transports.File({ filename: 'combined.log' }),
    ],
});

//
// If we're not in production then log to the `console` with the format:
// `${info.level}: ${info.message} JSON.stringify({ ...rest }) `
//
if (NODE_ENV !== 'production') {
    parentLogger.add(new winston.transports.Console({
        format: winston.format.simple(),
        level: lowest_level_to_report,
    }));
}

if (DOCKERIZED === true) {
    parentLogger.add(new winston.transports.Console({
        format: winston.format.simple(),
        level: lowest_level_to_report,
    }));
}

export { parentLogger };

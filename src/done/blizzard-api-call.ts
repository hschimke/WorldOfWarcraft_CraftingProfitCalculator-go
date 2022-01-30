import { EventEmitter } from 'events';
import got from 'got';
import { Logger } from 'winston';

function CPCApi(logging: Logger, api_auth: ApiAuthorization, config?: ApiConfig): CPCApi {
    const logger = logging;

    let allowed_connections_per_period = 100;
    let period_reset_window = 5000;

    if (config !== undefined) {
        if (config.connection_per_window !== undefined) {
            allowed_connections_per_period = config.connection_per_window;
        }
        if (config.window_size !== undefined) {
            period_reset_window = config.window_size;
        }
    }

    let allowed_during_period = 0;
    let in_use = 0;

    function sleep(ms: number): Promise<void> {
        return new Promise((resolve) => {
            setTimeout(resolve, ms);
        });
    }

    class BlizzardTimeoutManager extends EventEmitter {
        intervalTimeout?: NodeJS.Timeout;

        constructor() {
            super();
            this.intervalTimeout = undefined;
        }
    }
    const emitter = new BlizzardTimeoutManager();
    function exec_reset(): void {
        emitter.emit('reset');
    }

    function shutdownApiManager(): void {
        emitter.emit('shutdown');
    }

    emitter.on('reset', () => {
        logger.silly(`Resetting connection pool: used ${allowed_during_period} of available ${allowed_connections_per_period}, ${in_use} currently used`);
        allowed_during_period = 0 + in_use;
    });
    emitter.on('shutdown', () => {
        logger.debug(`Stop API manager with ${allowed_during_period} still running and ${in_use} in use`);
        if (emitter.intervalTimeout !== undefined) {
            clearInterval(emitter.intervalTimeout);
        }
    });
    emitter.on('start', () => {
        logger.debug(`Start API manager with window ${period_reset_window}`);
        emitter.intervalTimeout = setInterval(exec_reset, period_reset_window);
        emitter.intervalTimeout.unref();
    });

    async function manageBlizzardTimeout(): Promise<void> {
        emitter.emit('start');
    }

    const base_uri = 'api.blizzard.com';
    /**
     * Run a query against the blizzard api provider.
     * @param {!string} region_code The region code for the server.
     * @param {!Object} authorization_token An oath access token to use for the request.
     * @param {Object} data The request data to send.
     * @param {!string} uri The url to query against.
     */
    async function getBlizzardAPIResponse(region_code: RegionCode, data: string | Record<string, string | number>, uri: string): Promise<BlizzardApi.BlizzardApiReponse | void> {
        let proceed = false;
        let wait_count = 0;
        while (!proceed) {
            if (allowed_during_period > allowed_connections_per_period) {
                wait_count++;
                await sleep(1000);
            } else {
                proceed = true;
                allowed_during_period++;
            }
        }
        if (wait_count > 0) {
            logger.debug(`Waited ${wait_count} seconds for an available API window.`);
        }
        in_use++;
        try {
            const api_response = await got(`https://${region_code}.${base_uri}${uri}`, {
                responseType: 'json',
                method: 'GET',
                headers: {
                    'Connection': 'keep-alive',
                    'Authorization': `Bearer ${(await api_auth.getAuthorizationToken(region_code)).access_token}`
                },
                searchParams: data,
                //http2: true,
                dnsCache: true,
                retry: { limit: 5 },
                timeout: {
                    request: 10000
                }
            }).json();
            in_use--;
            return <BlizzardApi.BlizzardApiReponse>api_response;
        } catch (error) {
            logger.error(`Issue fetching blizzard data: (https://${region_code}.${base_uri}${uri}) ` + error);
        }
    }

    /**
     * 
     * @param {object} authorization_token An oath access token to use for the request.
     * @param {object} data The request data to send.
     * @param {string} uri Raw url including transport to query.
     */
    async function getBlizzardRawUriResponse(data: string | Record<string, string | number>, uri: string, region: RegionCode): Promise<BlizzardApi.BlizzardApiReponse | void> {
        let proceed = false;
        let wait_count = 0;
        while (!proceed) {
            if (allowed_during_period > allowed_connections_per_period) {
                wait_count++;
                await sleep(1000);
            } else {
                proceed = true;
                allowed_during_period++;
            }
        }
        if (wait_count > 0) {
            logger.debug(`Waited ${wait_count} seconds for an available API window.`);
        }
        in_use++;
        try {
            const api_response = await got(uri, {
                responseType: 'json',
                method: 'GET',
                headers: {
                    'Connection': 'keep-alive',
                    'Authorization': `Bearer ${(await api_auth.getAuthorizationToken(region)).access_token}`
                },
                searchParams: data,
                //http2: true,
                dnsCache: true,
                retry: { limit: 5 },
                timeout: {
                    request: 10000
                }
            }).json();
            in_use--;
            return <BlizzardApi.BlizzardApiReponse>api_response;
        } catch (error) {
            logger.error(`Issue fetching blizzard data: (${uri}) ` + error);
        }
    }

    manageBlizzardTimeout();

    return Object.freeze({
        getBlizzardAPIResponse,
        getBlizzardRawUriResponse,
        shutdownApiManager
    });
}

export { CPCApi };
